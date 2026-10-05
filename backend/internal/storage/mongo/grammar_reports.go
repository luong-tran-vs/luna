package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/grammar"
)

type grammarReportDoc struct {
	ID         bson.ObjectID `bson:"_id"`
	UserID     bson.ObjectID `bson:"userId"`
	PointID    string        `bson:"pointId"`
	ExerciseID string        `bson:"exerciseId"`
	Reason     string        `bson:"reason"`
	Note       string        `bson:"note"`
	Status     string        `bson:"status"`
	CreatedAt  time.Time     `bson:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt"`
	ResolvedAt time.Time     `bson:"resolvedAt,omitempty"`
}

func (d grammarReportDoc) toReport() grammar.Report {
	return grammar.Report{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), PointID: d.PointID, ExerciseID: d.ExerciseID,
		Reason: grammar.ReportReason(d.Reason), Note: d.Note, Status: grammar.ReportStatus(d.Status),
		CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC(), ResolvedAt: zeroOrUTC(d.ResolvedAt),
	}
}

// GrammarReports implements grammar.ReportRepository on "grammar_reports", one document per learner and exercise.
type GrammarReports struct {
	coll *mongo.Collection
}

// NewGrammarReports returns the grammar report repository.
func NewGrammarReports(db *mongo.Database) *GrammarReports {
	return &GrammarReports{coll: db.Collection("grammar_reports")}
}

var _ grammar.ReportRepository = (*GrammarReports)(nil)

// Upsert inserts the report, or reopens the learner's existing one with the new reason and note.
func (r *GrammarReports) Upsert(ctx context.Context, rep grammar.Report) error {
	uid, err := bson.ObjectIDFromHex(rep.UserID)
	if err != nil {
		return fmt.Errorf("grammar report user id: %w", err)
	}
	updated := rep.UpdatedAt
	if updated.IsZero() {
		updated = time.Now()
	}
	created := rep.CreatedAt
	if created.IsZero() {
		created = updated
	}
	_, err = r.coll.UpdateOne(ctx,
		bson.D{{Key: "userId", Value: uid}, {Key: "pointId", Value: rep.PointID}, {Key: "exerciseId", Value: rep.ExerciseID}},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "reason", Value: string(rep.Reason)}, {Key: "note", Value: rep.Note},
				{Key: "status", Value: string(grammar.ReportOpen)}, {Key: "updatedAt", Value: updated.UTC()},
			}},
			{Key: "$unset", Value: bson.D{{Key: "resolvedAt", Value: ""}}},
			{Key: "$setOnInsert", Value: bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "createdAt", Value: created.UTC()}}},
		}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("upsert grammar report: %w", err)
	}
	return nil
}

// ListOpen returns the open reports, newest first.
func (r *GrammarReports) ListOpen(ctx context.Context) ([]grammar.Report, error) {
	cur, err := r.coll.Find(ctx, bson.D{{Key: "status", Value: string(grammar.ReportOpen)}},
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "_id", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find grammar reports: %w", err)
	}
	var docs []grammarReportDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read grammar reports: %w", err)
	}
	out := make([]grammar.Report, len(docs))
	for i, d := range docs {
		out[i] = d.toReport()
	}
	return out, nil
}

// Resolve marks the open reports on the exercise resolved in one update and returns how many.
func (r *GrammarReports) Resolve(ctx context.Context, pointID, exerciseID string, at time.Time) (int, error) {
	res, err := r.coll.UpdateMany(ctx,
		bson.D{{Key: "pointId", Value: pointID}, {Key: "exerciseId", Value: exerciseID}, {Key: "status", Value: string(grammar.ReportOpen)}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: string(grammar.ReportResolved)}, {Key: "resolvedAt", Value: at.UTC()}}}})
	if err != nil {
		return 0, fmt.Errorf("resolve grammar reports: %w", err)
	}
	return int(res.ModifiedCount), nil
}
