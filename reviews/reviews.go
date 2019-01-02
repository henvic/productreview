package reviews

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"

	"github.com/henvic/productreview/db"
	"github.com/henvic/productreview/kv"
	log "github.com/sirupsen/logrus"
)

// Status of the review
type Status string

const (
	// ToReview queue command.
	ToReview = "to_review"

	// NotifyReviewed queue command.
	NotifyReviewed = "notify_reviewed"

	// Accepted review.
	Accepted Status = "accepted"

	// Rejected review.
	Rejected Status = "rejected"
)

// BadWords to be filtered.
var BadWords = []string{"fee", "nee", "cruul", "leent"}

// Review for a product.
type Review struct {
	ID        int    `db:"productreviewid"`
	ProductID int    `db:"productid"`
	Name      string `db:"reviewername"`
	Email     string `db:"emailaddress"`
	Review    string `db:"comments"`
	Rating    int    `db:"rating"`
	Status    Status `db:"status"`
}

// Response is sent when a product review is created.
type Response struct {
	Success  bool
	ReviewID int
}

// Create review
func Create(ctx context.Context, r Review) (id int, err error) {
	if err = Validate(r); err != nil {
		return 0, err
	}

	r.ID = rand.Intn(math.MaxInt32)

	conn := db.Conn()

	stmt, err := conn.PrepareContext(ctx, `
	INSERT INTO production.productreview (
		"productreviewid", "productid", "reviewername", "emailaddress", "comments", "rating")
		VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`)

	if err != nil {
		return 0, err
	}

	defer func() {
		if e := stmt.Close(); err == nil {
			err = e
		}
	}()

	var args = []interface{}{
		r.ID,
		r.ProductID,
		r.Name,
		r.Email,
		r.Review,
		r.Rating,
	}

	if _, err = stmt.ExecContext(ctx, args...); err != nil {
		log.Errorf("failed to add review %+v", r)
		return 0, err
	}

	re := kv.Conn()

	defer func() {
		if e := re.Close(); err == nil {
			err = e
		}
	}()

	if err = re.Send("RPUSH", ToReview, r.ID); err != nil {
		log.Errorf("failed to enqueue review %v to processor: %v", r.ID, err)
		return 0, err
	}

	return r.ID, err
}

// ValidationError for the review.
type ValidationError struct {
	msg string
}

func (v ValidationError) Error() string {
	return v.msg
}

// Validate review.
func Validate(r Review) error {
	if r.ProductID < 1 {
		return ValidationError{"invalid product ID"}
	}

	if len(r.Name) == 0 {
		return ValidationError{"name is empty"}
	}

	// TODO(henvic): use a validation library that follows RFC 5321
	if !strings.Contains(r.Email, "@") {
		return ValidationError{"invalid email address"}
	}

	if r.Rating < 0 || r.Rating > 5 {
		return ValidationError{"invalid rating value"}
	}

	return nil
}

// Get review.
func Get(ctx context.Context, id int) (r Review, err error) {
	var stmt *sqlx.Stmt

	conn := db.Conn()

	stmt, err = conn.PreparexContext(ctx, `SELECT
	productreviewid, productid, reviewername, emailaddress, comments, rating, status
	FROM production.productreview WHERE productreviewid = $1 LIMIT 1`)

	if err != nil {
		return r, err
	}

	defer func() {
		if e := stmt.Close(); err == nil {
			err = e
		}
	}()

	var rows *sqlx.Rows
	rows, err = stmt.QueryxContext(ctx, id)

	if err != nil {
		return r, err
	}

	if ok := rows.Next(); !ok {
		return r, sql.ErrNoRows
	}

	err = rows.StructScan(&r)
	return r, err
}

// Processor for the reviews.
func Processor(ctx context.Context) (err error) {
	re := kv.Conn()

	defer func() {
		e := re.Close()

		if err == nil {
			err = e
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			reply, err := redis.Values(re.Do("BLPOP", ToReview, 0))

			if err != nil {
				log.Error(err)
			}

			var ignore string
			var id int

			if _, err = redis.Scan(reply, &ignore, &id); err != nil {
				log.Error(err)
			}

			processReview(ctx, re, id)
		}
	}
}

func processReview(ctx context.Context, re redis.Conn, id int) {
	ctxReview, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var _, err = Verify(ctxReview, id)

	if err != nil {
		log.Errorf("error verifying review %v: %v", id, err)
		return
	}
}

// Notifier for the reviews.
func Notifier(ctx context.Context) (err error) {
	re := kv.Conn()

	defer func() {
		e := re.Close()

		if err == nil {
			err = e
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			reply, err := redis.Values(re.Do("BLPOP", NotifyReviewed, 0))

			if err != nil {
				log.Error(err)
			}

			var ignore string
			var id int

			if _, err = redis.Scan(reply, &ignore, &id); err != nil {
				log.Error(err)
			}

			processNotify(ctx, re, id)
		}
	}
}

func processNotify(ctx context.Context, re redis.Conn, id int) {
	r, err := Get(ctx, id)

	if err != nil {
		log.Error(err)
	}

	fmt.Printf("Email to %v: review %v was %v\n", r.Email, r.ID, r.Status)
}

// Verify review. Returns updated review.
func Verify(ctx context.Context, id int) (Status, error) {
	r, err := Get(ctx, id)

	if err != nil {
		return "", err
	}

	var status = Accepted

	var comments = strings.ToLower(r.Review)

	for _, b := range BadWords {
		// TODO(henvic): write regex to filter by a-z and split words
		// not considering effect on similar words (e.g., 'feel')
		if strings.Contains(comments, b) {
			status = Rejected
			break
		}
	}

	log.Debugf("flagging review %v as %v", id, status)

	if err := flagReviewStatus(ctx, id, status); err != nil {
		log.Errorf("cannot process review %v: %v", id, err)
		return status, err
	}

	r.Status = status

	notifyReviewed(r.ID)
	return status, nil
}

func notifyReviewed(id int) {
	re := kv.Conn()

	defer func() {
		if e := re.Close(); e != nil {
			log.Error(e)
		}
	}()

	if err := re.Send("RPUSH", NotifyReviewed, id); err != nil {
		log.Errorf("cannot enqueue notification for review %v", id)
	}
}

func flagReviewStatus(ctx context.Context, id int, status Status) error {
	conn := db.Conn()

	stmt, err := conn.PreparexContext(ctx, `UPDATE production.productreview
	SET status = $1
	WHERE productreviewid = $2`)

	if err != nil {
		return err
	}

	defer func() {
		if e := stmt.Close(); err == nil {
			err = e
		}
	}()

	var res sql.Result
	res, err = stmt.ExecContext(ctx, status, id)

	if err != nil {
		return err
	}

	var updated int64
	updated, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if updated != 1 {
		return fmt.Errorf("error updating review (%d rows modified)", updated)
	}

	return nil
}
