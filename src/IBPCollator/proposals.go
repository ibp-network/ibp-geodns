// -----------------------------------------------------------------------------
// src/IBPCollator/proposals.go                                         ░░ IBP ░░
// -----------------------------------------------------------------------------
//
//  Persistence helpers for proposals.
//
//  By default we try to write every proposal (and its final result) to the
//  `proposals` table.  When that table is not present, we *silently ignore*
//  MySQL error 1146 so the collator can run completely stateless.
//
//  If you later decide you want persistence again, just create the table
//  and the same binary will start writing to it automatically.
// -----------------------------------------------------------------------------

package main

import (
	"database/sql"

	mysql "github.com/go-sql-driver/mysql"

	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
)

// UpsertProposal adds the proposal if it is new or leaves it untouched if it
// already exists.  When the backing table is missing the function is a no‑op.
func UpsertProposal(db *sql.DB, p data2.Proposal) {
	if db == nil {
		return
	}

	_, err := db.Exec(`
		INSERT INTO proposals   (id, finalised, passed)
		             VALUES     (?,  0,        NULL)
		ON DUPLICATE KEY UPDATE id = id
	`, p.ID)

	if ignoreMissingTable(err) {
		return
	}
	if err != nil {
		log.Log(log.Error, "[collator] UpsertProposal: %v", err)
	}
}

// InsertProposal is kept for legacy callers that still use the old name.
func InsertProposal(db *sql.DB, p data2.Proposal) {
	UpsertProposal(db, p)
}

// MarkProposalFinal is invoked once a proposal has been agreed on.  If the
// table is absent we just skip the write and keep going.
func MarkProposalFinal(db *sql.DB, id data2.ProposalID, passed bool) {
	if db == nil {
		return
	}

	_, err := db.Exec(`
		UPDATE proposals
		   SET finalised = 1,
		       passed     = ?
		 WHERE id        = ?
	`, passed, id)

	if ignoreMissingTable(err) {
		return
	}
	if err != nil {
		log.Log(log.Error, "[collator] MarkProposalFinal: %v", err)
	}
}

// -----------------------------------------------------------------------------
// internal helpers
// -----------------------------------------------------------------------------

// ignoreMissingTable returns true when the error is the specific MySQL
// 1146 “table doesn’t exist” error produced if the user chose to run the
// collator without creating the `proposals` table.
func ignoreMissingTable(err error) bool {
	if err == nil {
		return false
	}
	if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1146 {
		// The user intentionally runs without the persistence table.
		return true
	}
	return false
}
