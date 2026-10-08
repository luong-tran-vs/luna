package mongo

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/wordbank"
)

// bankWordDoc is one word of the bank in "word_bank", keyed by its lemma.
type bankWordDoc struct {
	Lemma     string     `bson:"_id"`
	MeaningVi string     `bson:"meaningVi"`
	IPA       string     `bson:"ipa"`
	ImageAt   *time.Time `bson:"imageAt"`
	CreatedAt time.Time  `bson:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt"`
}

// bankImageDoc is the picture of a word in "word_bank_images", keyed by its lemma.
type bankImageDoc struct {
	Lemma     string    `bson:"_id"`
	MIME      string    `bson:"mime"`
	Data      []byte    `bson:"data"`
	CreatedAt time.Time `bson:"createdAt"`
}

func (d bankWordDoc) toWord() wordbank.Word {
	w := wordbank.Word{Lemma: d.Lemma, MeaningVi: d.MeaningVi, IPA: d.IPA, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
	if d.ImageAt != nil {
		w.ImageAt = *d.ImageAt
	}
	return w
}

func toBankWordDoc(w wordbank.Word) bankWordDoc {
	return bankWordDoc{Lemma: w.Lemma, MeaningVi: w.MeaningVi, IPA: w.IPA, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt}
}

// WordBank implements wordbank.Repository (F24): the words in "word_bank" and their pictures, as
// binary data, in "word_bank_images".
type WordBank struct {
	words  *mongo.Collection
	images *mongo.Collection
}

// NewWordBank returns the word bank repository on db.
func NewWordBank(db *mongo.Database) *WordBank {
	return &WordBank{words: db.Collection("word_bank"), images: db.Collection("word_bank_images")}
}

var _ wordbank.Repository = (*WordBank)(nil)

// List returns a page of the matching words by lemma, and how many match.
func (r *WordBank) List(ctx context.Context, q wordbank.ListQuery) ([]wordbank.Word, int, error) {
	filter := bson.D{}
	if q.Search != "" {
		re := bson.Regex{Pattern: regexp.QuoteMeta(q.Search), Options: "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "_id", Value: re}}, bson.D{{Key: "meaningVi", Value: re}},
		}})
	}
	switch q.Missing {
	case wordbank.MissingImage:
		filter = append(filter, bson.E{Key: "imageAt", Value: nil})
	case wordbank.MissingIPA:
		filter = append(filter, bson.E{Key: "ipa", Value: ""})
	case wordbank.MissingMeaning:
		filter = append(filter, bson.E{Key: "meaningVi", Value: ""})
	}
	if q.Lemmas != nil {
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$in", Value: q.Lemmas}}})
	}
	total, err := r.words.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count bank words: %w", err)
	}
	opts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetSkip(int64(q.Skip)).SetLimit(int64(q.Limit))
	words, err := r.findMany(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	return words, int(total), nil
}

func (r *WordBank) findMany(ctx context.Context, filter bson.D, opts *options.FindOptionsBuilder) ([]wordbank.Word, error) {
	cur, err := r.words.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find bank words: %w", err)
	}
	var docs []bankWordDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read bank words: %w", err)
	}
	out := make([]wordbank.Word, len(docs))
	for i, d := range docs {
		out[i] = d.toWord()
	}
	return out, nil
}

// Get returns wordbank.ErrNotFound when the bank has no such word.
func (r *WordBank) Get(ctx context.Context, lemma string) (wordbank.Word, error) {
	var d bankWordDoc
	err := r.words.FindOne(ctx, bson.D{{Key: "_id", Value: lemma}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return wordbank.Word{}, wordbank.ErrNotFound
	}
	if err != nil {
		return wordbank.Word{}, fmt.Errorf("find bank word: %w", err)
	}
	return d.toWord(), nil
}

// Find returns the words of the bank among lemmas.
func (r *WordBank) Find(ctx context.Context, lemmas []string) ([]wordbank.Word, error) {
	if len(lemmas) == 0 {
		return nil, nil
	}
	return r.findMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: lemmas}}}}, options.Find())
}

// Create returns wordbank.ErrExists when the lemma is taken.
func (r *WordBank) Create(ctx context.Context, w wordbank.Word) error {
	_, err := r.words.InsertOne(ctx, toBankWordDoc(w))
	if mongo.IsDuplicateKeyError(err) {
		return wordbank.ErrExists
	}
	if err != nil {
		return fmt.Errorf("insert bank word: %w", err)
	}
	return nil
}

// InsertMissing adds the words not in the bank yet; words added meanwhile are skipped.
func (r *WordBank) InsertMissing(ctx context.Context, ws []wordbank.Word) (int, error) {
	if len(ws) == 0 {
		return 0, nil
	}
	docs := make([]any, len(ws))
	for i, w := range ws {
		docs[i] = toBankWordDoc(w)
	}
	res, err := r.words.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	if err != nil && !onlyDuplicates(err) {
		return 0, fmt.Errorf("insert bank words: %w", err)
	}
	if res == nil {
		return 0, nil
	}
	return len(res.InsertedIDs), nil
}

// onlyDuplicates says every failed write of a bulk insert hit an existing key.
func onlyDuplicates(err error) bool {
	var bwe mongo.BulkWriteException
	if !errors.As(err, &bwe) || bwe.WriteConcernError != nil {
		return false
	}
	for _, we := range bwe.WriteErrors {
		if we.Code != 11000 { // duplicate key
			return false
		}
	}
	return true
}

// Update sets the meaning, IPA and update time of a word.
func (r *WordBank) Update(ctx context.Context, w wordbank.Word) error {
	res, err := r.words.UpdateOne(ctx, bson.D{{Key: "_id", Value: w.Lemma}}, bson.D{{Key: "$set", Value: bson.D{
		{Key: "meaningVi", Value: w.MeaningVi}, {Key: "ipa", Value: w.IPA}, {Key: "updatedAt", Value: w.UpdatedAt},
	}}})
	if err != nil {
		return fmt.Errorf("update bank word: %w", err)
	}
	if res.MatchedCount == 0 {
		return wordbank.ErrNotFound
	}
	return nil
}

// Delete removes a word and its picture.
func (r *WordBank) Delete(ctx context.Context, lemma string) error {
	res, err := r.words.DeleteOne(ctx, bson.D{{Key: "_id", Value: lemma}})
	if err != nil {
		return fmt.Errorf("delete bank word: %w", err)
	}
	if res.DeletedCount == 0 {
		return wordbank.ErrNotFound
	}
	if _, err := r.images.DeleteOne(ctx, bson.D{{Key: "_id", Value: lemma}}); err != nil {
		return fmt.Errorf("delete bank image: %w", err)
	}
	return nil
}

// Image returns wordbank.ErrImageNotFound when the word has no picture.
func (r *WordBank) Image(ctx context.Context, lemma string) (wordbank.Image, error) {
	var d bankImageDoc
	err := r.images.FindOne(ctx, bson.D{{Key: "_id", Value: lemma}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return wordbank.Image{}, wordbank.ErrImageNotFound
	}
	if err != nil {
		return wordbank.Image{}, fmt.Errorf("find bank image: %w", err)
	}
	return wordbank.Image{Lemma: d.Lemma, MIME: d.MIME, Data: d.Data, CreatedAt: d.CreatedAt}, nil
}

// SaveImage replaces the picture of a word, then marks the word as having one.
func (r *WordBank) SaveImage(ctx context.Context, img wordbank.Image) error {
	if _, err := r.Get(ctx, img.Lemma); err != nil {
		return err
	}
	d := bankImageDoc{Lemma: img.Lemma, MIME: img.MIME, Data: img.Data, CreatedAt: img.CreatedAt}
	if _, err := r.images.ReplaceOne(ctx, bson.D{{Key: "_id", Value: img.Lemma}}, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save bank image: %w", err)
	}
	res, err := r.words.UpdateOne(ctx, bson.D{{Key: "_id", Value: img.Lemma}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "imageAt", Value: img.CreatedAt}}}})
	if err != nil {
		return fmt.Errorf("mark bank image: %w", err)
	}
	if res.MatchedCount == 0 { // the word was deleted meanwhile
		_, _ = r.images.DeleteOne(ctx, bson.D{{Key: "_id", Value: img.Lemma}})
		return wordbank.ErrNotFound
	}
	return nil
}

// DeleteImage removes the picture of a word.
func (r *WordBank) DeleteImage(ctx context.Context, lemma string) error {
	if _, err := r.words.UpdateOne(ctx, bson.D{{Key: "_id", Value: lemma}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "imageAt", Value: nil}}}}); err != nil {
		return fmt.Errorf("unmark bank image: %w", err)
	}
	if _, err := r.images.DeleteOne(ctx, bson.D{{Key: "_id", Value: lemma}}); err != nil {
		return fmt.Errorf("delete bank image: %w", err)
	}
	return nil
}
