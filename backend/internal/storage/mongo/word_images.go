package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// imageSettingsDoc is one lesson's picture settings in "lesson_image_settings", keyed by lesson id.
type imageSettingsDoc struct {
	LessonID  bson.ObjectID `bson:"_id"`
	Enabled   bool          `bson:"enabled"`
	Style     string        `bson:"style"`
	Status    string        `bson:"status"`
	Error     string        `bson:"error"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// wordImageDoc is one word picture in "word_images", unique by (lessonId, lemma).
type wordImageDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	LessonID  bson.ObjectID `bson:"lessonId"`
	Lemma     string        `bson:"lemma"`
	MIME      string        `bson:"mime"`
	Data      []byte        `bson:"data"`
	CreatedAt time.Time     `bson:"createdAt"`
}

func (d imageSettingsDoc) toSettings() lesson.ImageSettings {
	return lesson.ImageSettings{
		Enabled: d.Enabled, Style: d.Style, Status: lesson.Status(d.Status), Error: d.Error, UpdatedAt: d.UpdatedAt,
	}
}

// WordImages implements lesson.ImageRepository (F23): the picture settings of each lesson in
// "lesson_image_settings" and the pictures, as binary data, in "word_images".
type WordImages struct {
	settings *mongo.Collection
	images   *mongo.Collection
}

// NewWordImages returns the word picture repository on db.
func NewWordImages(db *mongo.Database) *WordImages {
	return &WordImages{settings: db.Collection("lesson_image_settings"), images: db.Collection("word_images")}
}

var _ lesson.ImageRepository = (*WordImages)(nil)

// Settings returns the zero settings for a lesson without any (or a malformed id).
func (r *WordImages) Settings(ctx context.Context, lessonID string) (lesson.ImageSettings, error) {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return lesson.ImageSettings{}, nil //nolint:nilerr // a malformed id has no settings
	}
	var d imageSettingsDoc
	err = r.settings.FindOne(ctx, bson.D{{Key: "_id", Value: lid}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return lesson.ImageSettings{}, nil
	}
	if err != nil {
		return lesson.ImageSettings{}, fmt.Errorf("find image settings: %w", err)
	}
	return d.toSettings(), nil
}

// SaveSettings replaces the settings of a lesson.
func (r *WordImages) SaveSettings(ctx context.Context, lessonID string, s lesson.ImageSettings) error {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return lesson.ErrNotFound
	}
	d := imageSettingsDoc{
		LessonID: lid, Enabled: s.Enabled, Style: s.Style, Status: string(s.Status), Error: s.Error,
		UpdatedAt: s.UpdatedAt.UTC(),
	}
	_, err = r.settings.ReplaceOne(ctx, bson.D{{Key: "_id", Value: lid}}, d, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save image settings: %w", err)
	}
	return nil
}

// Lemmas lists the words of a lesson that have a picture.
func (r *WordImages) Lemmas(ctx context.Context, lessonID string) ([]string, error) {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return []string{}, nil //nolint:nilerr // a malformed id has no pictures
	}
	cur, err := r.images.Find(ctx, bson.D{{Key: "lessonId", Value: lid}},
		options.Find().SetProjection(bson.D{{Key: "lemma", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find word images: %w", err)
	}
	var docs []struct {
		Lemma string `bson:"lemma"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode word images: %w", err)
	}
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = d.Lemma
	}
	return out, nil
}

// Get returns lesson.ErrImageNotFound when the word has no picture.
func (r *WordImages) Get(ctx context.Context, lessonID, lemma string) (lesson.WordImage, error) {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return lesson.WordImage{}, lesson.ErrImageNotFound
	}
	var d wordImageDoc
	err = r.images.FindOne(ctx, bson.D{{Key: "lessonId", Value: lid}, {Key: "lemma", Value: lemma}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return lesson.WordImage{}, lesson.ErrImageNotFound
	}
	if err != nil {
		return lesson.WordImage{}, fmt.Errorf("find word image: %w", err)
	}
	return lesson.WordImage{
		LessonID: lessonID, Lemma: d.Lemma, MIME: d.MIME, Data: d.Data, CreatedAt: d.CreatedAt,
	}, nil
}

// Save stores a picture, replacing the word's previous one.
func (r *WordImages) Save(ctx context.Context, img lesson.WordImage) error {
	lid, err := bson.ObjectIDFromHex(img.LessonID)
	if err != nil {
		return lesson.ErrNotFound
	}
	filter := bson.D{{Key: "lessonId", Value: lid}, {Key: "lemma", Value: img.Lemma}}
	d := wordImageDoc{LessonID: lid, Lemma: img.Lemma, MIME: img.MIME, Data: img.Data, CreatedAt: img.CreatedAt.UTC()}
	if _, err := r.images.ReplaceOne(ctx, filter, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save word image: %w", err)
	}
	return nil
}

// Delete removes the picture of one word.
func (r *WordImages) Delete(ctx context.Context, lessonID, lemma string) error {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return nil //nolint:nilerr // a malformed id has nothing stored
	}
	if _, err := r.images.DeleteOne(ctx, bson.D{{Key: "lessonId", Value: lid}, {Key: "lemma", Value: lemma}}); err != nil {
		return fmt.Errorf("delete word image: %w", err)
	}
	return nil
}

// DeleteForLesson removes the settings and every picture of a lesson.
func (r *WordImages) DeleteForLesson(ctx context.Context, lessonID string) error {
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return nil //nolint:nilerr // a malformed id has nothing stored
	}
	if _, err := r.images.DeleteMany(ctx, bson.D{{Key: "lessonId", Value: lid}}); err != nil {
		return fmt.Errorf("delete word images: %w", err)
	}
	if _, err := r.settings.DeleteOne(ctx, bson.D{{Key: "_id", Value: lid}}); err != nil {
		return fmt.Errorf("delete image settings: %w", err)
	}
	return nil
}
