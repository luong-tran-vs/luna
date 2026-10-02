package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// Practice documents (F17) are embedded in the lesson document.

type exampleDoc struct {
	Lemma    string `bson:"lemma"`
	Sentence string `bson:"sentence"`
}

type turnDoc struct {
	Speaker   int    `bson:"speaker"`
	Text      string `bson:"text"`
	MeaningVi string `bson:"meaningVi"`
}

type dialogueDoc struct {
	Speakers []string  `bson:"speakers"`
	Turns    []turnDoc `bson:"turns"`
}

type translationDoc struct {
	Vi          string   `bson:"vi"`
	En          string   `bson:"en"`
	Distractors []string `bson:"distractors"`
}

type practiceDoc struct {
	ObjectiveVi  string           `bson:"objectiveVi"`
	Examples     []exampleDoc     `bson:"examples"`
	Dialogue     *dialogueDoc     `bson:"dialogue"`
	GrammarTipVi string           `bson:"grammarTipVi"`
	Translations []translationDoc `bson:"translations"`
}

func fromPractice(p *lesson.Practice) *practiceDoc {
	if p == nil {
		return nil
	}
	d := &practiceDoc{
		ObjectiveVi:  p.ObjectiveVi,
		GrammarTipVi: p.GrammarTipVi,
		Examples:     make([]exampleDoc, len(p.Examples)),
		Translations: make([]translationDoc, len(p.Translations)),
	}
	for i, e := range p.Examples {
		d.Examples[i] = exampleDoc(e)
	}
	for i, t := range p.Translations {
		d.Translations[i] = translationDoc(t)
	}
	if p.Dialogue != nil {
		d.Dialogue = &dialogueDoc{Speakers: p.Dialogue.Speakers, Turns: make([]turnDoc, len(p.Dialogue.Turns))}
		for i, t := range p.Dialogue.Turns {
			d.Dialogue.Turns[i] = turnDoc(t)
		}
	}
	return d
}

func (d *practiceDoc) toPractice() *lesson.Practice {
	if d == nil {
		return nil
	}
	p := &lesson.Practice{
		ObjectiveVi:  d.ObjectiveVi,
		GrammarTipVi: d.GrammarTipVi,
		Examples:     make([]lesson.Example, len(d.Examples)),
		Translations: make([]lesson.Translation, len(d.Translations)),
	}
	for i, e := range d.Examples {
		p.Examples[i] = lesson.Example(e)
	}
	for i, t := range d.Translations {
		p.Translations[i] = lesson.Translation(t)
	}
	if d.Dialogue != nil {
		p.Dialogue = &lesson.Dialogue{Speakers: d.Dialogue.Speakers, Turns: make([]lesson.Turn, len(d.Dialogue.Turns))}
		for i, t := range d.Dialogue.Turns {
			p.Dialogue.Turns[i] = lesson.Turn(t)
		}
	}
	return p
}

// SavePractice stores a practice, marks it done and bumps practiceVersion, if the lesson is
// still at revision and practiceVersion still equals prevVersion (lessons from before F17 have
// no practiceVersion, which counts as 0).
func (r *Lessons) SavePractice(ctx context.Context, id string, revision, prevVersion int, p lesson.Practice) (bool, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, lesson.ErrNotFound
	}
	var version any = prevVersion
	if prevVersion == 0 {
		version = bson.D{{Key: "$in", Value: bson.A{0, nil}}}
	}
	filter := bson.D{{Key: "_id", Value: oid}, {Key: "revision", Value: revision}, {Key: "practiceVersion", Value: version}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "practice", Value: fromPractice(&p)},
			{Key: "practiceStatus", Value: string(lesson.StatusDone)},
			{Key: "practiceError", Value: ""},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}},
		{Key: "$inc", Value: bson.D{{Key: "practiceVersion", Value: 1}}},
	}
	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("save practice: %w", err)
	}
	return res.MatchedCount == 1, nil
}

// WithoutPractice lists the lessons whose annotations are done but whose practice status was
// never set: lessons annotated before F17 have no practiceStatus field at all.
func (r *Lessons) WithoutPractice(ctx context.Context) ([]lesson.RevisionRef, error) {
	filter := bson.D{
		{Key: "annotationStatus", Value: string(lesson.StatusDone)},
		{Key: "practiceStatus", Value: bson.D{{Key: "$in", Value: bson.A{nil, string(lesson.StatusNone)}}}},
	}
	opts := options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}, {Key: "revision", Value: 1}})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find lessons without practice: %w", err)
	}
	var docs []struct {
		ID       bson.ObjectID `bson:"_id"`
		Revision int           `bson:"revision"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read lessons without practice: %w", err)
	}
	out := make([]lesson.RevisionRef, len(docs))
	for i, d := range docs {
		out[i] = lesson.RevisionRef{ID: d.ID.Hex(), Revision: d.Revision}
	}
	return out, nil
}
