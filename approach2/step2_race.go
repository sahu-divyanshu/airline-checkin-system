// ============================================================
// Step 3 — THE EXCLUSIVE LOCK (FOR UPDATE)
// ============================================================
//
// Same 500 goroutines. This time the SELECT and UPDATE are
// wrapped in a single transaction with FOR UPDATE.
//
// How it works:
//   BEGIN
//   SELECT * FROM seats WHERE id=1 AND status='available' FOR UPDATE
//      → Worker 1 acquires a row-level exclusive lock
//      → Workers 2–500 BLOCK here waiting for the lock
//   UPDATE seats SET status='booked' WHERE id=1
//   COMMIT  → lock released
//      → Workers 2–500 unblock, re-read the row, see 'booked', skip
//
// Exactly one worker books the seat. 499 fail correctly.
// The database serialises the race for you.
//
// Run:
//   go run step3_for_update.go
// ============================================================

package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dsn        = "postgres://postgres:strong_password@127.0.0.1:5432/airline_db"
	numWorkers = 100
)

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn+"?pool_max_conns=200")
	if err != nil {
		panic(fmt.Sprintf("cannot connect to DB: %v", err))
	}
	defer pool.Close()

	// Reset seat 1
	_, err = pool.Exec(ctx, "UPDATE seats SET status='available', user_id=NULL;")
	if err != nil {
		panic(err)
	}
	fmt.Println("✔  Seat 1 reset to 'available'")
	fmt.Printf("🚀 Launching %d concurrent workers with FOR UPDATE locking...\n\n", numWorkers)

	var (
		booked  atomic.Int64
		skipped atomic.Int64
		errors  atomic.Int64
		wg      sync.WaitGroup
	)

	start := time.Now()
	_, err = pool.Exec(ctx, "UPDATE seats SET status='available', user_id=NULL;")
	if err != nil {
		panic(err)
	}
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// ── Transaction wraps SELECT + UPDATE atomically ─────────────────
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{
				IsoLevel: pgx.ReadCommitted, // standard isolation level
			})
			if err != nil {
				log.Println("BeginTx me error aa gaya bhai", err.Error())
				errors.Add(1)
				return
			}

			// ── FOR UPDATE acquires a row-level exclusive lock ───────────────
			// If another transaction already holds the lock, THIS LINE BLOCKS
			// until that transaction commits or rolls back.
			//
			// AND status='available' means: even after unblocking and
			// re-reading, if status is now 'booked' we get 0 rows back
			// (the seat was already taken) — so we skip cleanly.
			var seatID int
			err = tx.QueryRow(ctx, `
				SELECT id 
				FROM seats 
				WHERE status = 'available' 
				ORDER BY id 
				LIMIT 1;
			`).Scan(&seatID)
			if err == pgx.ErrNoRows {
				_ = tx.Rollback(ctx)
				log.Println("Select QueryRow me kuch nahi aya bhai", err.Error())
				skipped.Add(1)
				return
			}
			if err != nil {
				_ = tx.Rollback(ctx)
				log.Println("Select QueryRow me error aa gaya bhai", err.Error())
				errors.Add(1)
				return
			}
			_, err = tx.Exec(ctx, `
				    UPDATE seats
				    SET status  = 'booked',
				        user_id = $1
				    WHERE  id = $2
				`, i, seatID)
			if err != nil {
				log.Println("Update Exec me error aa gaya bhai", err.Error())
				_ = tx.Rollback(ctx)
				errors.Add(1)
				return
			}

			if err = tx.Commit(ctx); err != nil {
				log.Println("commit me error aa gaya bhai", err.Error())
				errors.Add(1)
				return
			}

			booked.Add(1)
			fmt.Printf("  Worker %03d → BOOKED  ✓  (seat %d)\n", workerID, seatID)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Println("\n" + "═════════════════════════════════════════")
	fmt.Printf("  STEP 3 RESULTS — Race Condition\n")
	fmt.Printf("  Duration : %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Workers  : %d\n", numWorkers)
	fmt.Printf("  Booked   : %d  ← must be exactly 1\n", booked.Load())
	fmt.Printf("  Skipped  : %d  ← correctly rejected\n", skipped.Load())
	fmt.Printf("  Errors   : %d\n", errors.Load())
	fmt.Println("═════════════════════════════════════════")

	if booked.Load() == 1 && skipped.Load() == numWorkers-1 {
		fmt.Println("\n✅  PERFECT — exactly one seat booked, all others correctly rejected.")
	} else {
		fmt.Printf("\n🔴  Unexpected result — booked=%d, check your DB state.\n", booked.Load())
	}

	fmt.Println("\n── Verify in psql ──────────────────────────────────────────")
	fmt.Println("   SELECT * FROM seats WHERE id=1;")
}
