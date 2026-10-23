package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/curriculum"
)

// Curricula implements curriculum.Repository on "curricula" (added 2026-10-23): one document per
// level, its _id the level, with the approved course and the draft embedded.
type Curricula struct {
	coll *mongo.Collection
}

// NewCurricula returns the Curricula repository on db.
func NewCurricula(db *mongo.Database) *Curricula {
	return &Curricula{coll: db.Collection("curricula")}
}

var _ curriculum.Repository = (*Curricula)(nil)

type curriculumSlotDoc struct {
	Key     string   `bson:"key"`
	Kind    string   `bson:"kind"`
	Title   string   `bson:"title"`
	Grammar []string `bson:"grammar"`
	Idea    string   `bson:"idea"`
}

type curriculumTopicDoc struct {
	TopicID string              `bson:"topicId,omitempty"`
	Name    string              `bson:"name"`
	CanDoVi string              `bson:"canDoVi"`
	Grammar []string            `bson:"grammar"`
	Slots   []curriculumSlotDoc `bson:"slots"`
}

type curriculumPlanDoc struct {
	Topics []curriculumTopicDoc `bson:"topics"`
}

type curriculumDoc struct {
	Level      string             `bson:"_id"`
	Approved   *curriculumPlanDoc `bson:"approved"`
	Draft      *curriculumPlanDoc `bson:"draft"`
	ApprovedAt *time.Time         `bson:"approvedAt"`
	UpdatedAt  time.Time          `bson:"updatedAt"`
}

func toCurriculumPlanDoc(p *curriculum.Plan) *curriculumPlanDoc {
	if p == nil {
		return nil
	}
	out := &curriculumPlanDoc{Topics: make([]curriculumTopicDoc, len(p.Topics))}
	for i, t := range p.Topics {
		td := curriculumTopicDoc{TopicID: t.TopicID, Name: t.Name, CanDoVi: t.CanDoVi, Grammar: t.Grammar,
			Slots: make([]curriculumSlotDoc, len(t.Slots))}
		for j, s := range t.Slots {
			td.Slots[j] = curriculumSlotDoc{Key: s.Key, Kind: string(s.Kind), Title: s.Title, Grammar: s.Grammar, Idea: s.Idea}
		}
		out.Topics[i] = td
	}
	return out
}

func (d *curriculumPlanDoc) plan(level string) *curriculum.Plan {
	if d == nil {
		return nil
	}
	out := &curriculum.Plan{Level: level, Topics: make([]curriculum.Topic, len(d.Topics))}
	for i, t := range d.Topics {
		top := curriculum.Topic{TopicID: t.TopicID, Name: t.Name, CanDoVi: t.CanDoVi, Grammar: t.Grammar,
			Slots: make([]curriculum.Slot, len(t.Slots))}
		for j, s := range t.Slots {
			top.Slots[j] = curriculum.Slot{Key: s.Key, Kind: curriculum.Kind(s.Kind), Title: s.Title, Grammar: s.Grammar, Idea: s.Idea}
		}
		out.Topics[i] = top
	}
	return out
}

// Get returns the course of level, or curriculum.ErrNoPlan.
func (r *Curricula) Get(ctx context.Context, level string) (curriculum.Course, error) {
	var d curriculumDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: level}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return curriculum.Course{}, curriculum.ErrNoPlan
	}
	if err != nil {
		return curriculum.Course{}, fmt.Errorf("mongo: get curriculum: %w", err)
	}
	c := curriculum.Course{Level: d.Level, Approved: d.Approved.plan(level), Draft: d.Draft.plan(level), UpdatedAt: d.UpdatedAt}
	if d.ApprovedAt != nil {
		c.ApprovedAt = *d.ApprovedAt
	}
	return c, nil
}

// Save writes the course of c.Level, replacing it.
func (r *Curricula) Save(ctx context.Context, c curriculum.Course) error {
	d := curriculumDoc{
		Level: c.Level, Approved: toCurriculumPlanDoc(c.Approved), Draft: toCurriculumPlanDoc(c.Draft),
		UpdatedAt: c.UpdatedAt.UTC(),
	}
	if !c.ApprovedAt.IsZero() {
		at := c.ApprovedAt.UTC()
		d.ApprovedAt = &at
	}
	_, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: c.Level}}, d, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("mongo: save curriculum: %w", err)
	}
	return nil
}
