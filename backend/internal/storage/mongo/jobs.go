package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/job"
)

type jobDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Type      string        `bson:"type"`
	LessonID  bson.ObjectID `bson:"lessonId"`
	Revision  int           `bson:"revision"`
	Status    string        `bson:"status"`
	Attempts  int           `bson:"attempts"`
	Error     string        `bson:"error"`
	RunAt     time.Time     `bson:"runAt"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

func (d jobDoc) toJob() job.Job {
	return job.Job{
		ID: d.ID.Hex(), Type: job.Type(d.Type), LessonID: d.LessonID.Hex(), Revision: d.Revision,
		Status: job.Status(d.Status), Attempts: d.Attempts, Error: d.Error,
		RunAt: d.RunAt, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// Jobs implements job.Repository on the "jobs" collection.
type Jobs struct {
	coll *mongo.Collection
}

// NewJobs returns a job repository on db.
func NewJobs(db *mongo.Database) *Jobs {
	return &Jobs{coll: db.Collection("jobs")}
}

var _ job.Repository = (*Jobs)(nil)

// Enqueue inserts a pending job.
func (r *Jobs) Enqueue(ctx context.Context, j job.Job) error {
	lid, err := bson.ObjectIDFromHex(j.LessonID)
	if err != nil {
		return fmt.Errorf("job lesson id: %w", err)
	}
	now := time.Now().UTC()
	status := j.Status
	if status == "" {
		status = job.StatusPending
	}
	d := jobDoc{
		ID: bson.NewObjectID(), Type: string(j.Type), LessonID: lid, Revision: j.Revision,
		Status: string(status), RunAt: j.RunAt.UTC(), CreatedAt: now, UpdatedAt: now,
	}
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

// ClaimNext atomically takes the oldest due pending job.
func (r *Jobs) ClaimNext(ctx context.Context, now time.Time) (job.Job, bool, error) {
	var d jobDoc
	err := r.coll.FindOneAndUpdate(ctx,
		bson.D{{Key: "status", Value: string(job.StatusPending)}, {Key: "runAt", Value: bson.D{{Key: "$lte", Value: now.UTC()}}}},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "status", Value: string(job.StatusRunning)}, {Key: "updatedAt", Value: now.UTC()}}},
			{Key: "$inc", Value: bson.D{{Key: "attempts", Value: 1}}},
		},
		options.FindOneAndUpdate().SetSort(bson.D{{Key: "runAt", Value: 1}}).SetReturnDocument(options.After),
	).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return job.Job{}, false, nil
	}
	if err != nil {
		return job.Job{}, false, fmt.Errorf("claim job: %w", err)
	}
	return d.toJob(), true, nil
}

// Complete marks a job done.
func (r *Jobs) Complete(ctx context.Context, id string) error {
	return r.set(ctx, id, bson.D{{Key: "status", Value: string(job.StatusDone)}, {Key: "error", Value: ""}})
}

// Retry puts a job back to pending from runAt.
func (r *Jobs) Retry(ctx context.Context, id string, runAt time.Time, errMsg string) error {
	return r.set(ctx, id, bson.D{
		{Key: "status", Value: string(job.StatusPending)}, {Key: "runAt", Value: runAt.UTC()}, {Key: "error", Value: errMsg},
	})
}

// Fail marks a job failed.
func (r *Jobs) Fail(ctx context.Context, id, errMsg string) error {
	return r.set(ctx, id, bson.D{{Key: "status", Value: string(job.StatusFailed)}, {Key: "error", Value: errMsg}})
}

// DeletePending drops queued (not running) jobs of a lesson.
func (r *Jobs) DeletePending(ctx context.Context, lessonID string) error {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return fmt.Errorf("job lesson id: %w", err)
	}
	_, err = r.coll.DeleteMany(ctx, bson.D{{Key: "lessonId", Value: lid}, {Key: "status", Value: string(job.StatusPending)}})
	if err != nil {
		return fmt.Errorf("delete pending jobs: %w", err)
	}
	return nil
}

// DeleteForLesson drops every job of a lesson.
func (r *Jobs) DeleteForLesson(ctx context.Context, lessonID string) error {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return fmt.Errorf("job lesson id: %w", err)
	}
	if _, err := r.coll.DeleteMany(ctx, bson.D{{Key: "lessonId", Value: lid}}); err != nil {
		return fmt.Errorf("delete jobs: %w", err)
	}
	return nil
}

// ResetRunning returns jobs left running by a previous process to pending.
func (r *Jobs) ResetRunning(ctx context.Context) (int64, error) {
	res, err := r.coll.UpdateMany(ctx,
		bson.D{{Key: "status", Value: string(job.StatusRunning)}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: string(job.StatusPending)}, {Key: "updatedAt", Value: time.Now().UTC()}}}})
	if err != nil {
		return 0, fmt.Errorf("reset running jobs: %w", err)
	}
	return res.ModifiedCount, nil
}

func (r *Jobs) set(ctx context.Context, id string, set bson.D) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("job id: %w", err)
	}
	set = append(set, bson.E{Key: "updatedAt", Value: time.Now().UTC()})
	if _, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: oid}}, bson.D{{Key: "$set", Value: set}}); err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	return nil
}
