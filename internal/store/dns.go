package store

import (
	"database/sql"
	"fmt"
	"time"
)

type DNSRecord struct {
	ID        int
	Owner     string
	Target    string
	DomainID  int
	Domain    string // joined from domains table
	CreatedBy string
	UpdatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func DNSAdd(db *sql.DB, owner, target, user string) error {
	domain, err := FindParentDomain(db, owner)
	if err != nil {
		return err
	}

	if IsDomainExpired(domain) {
		return fmt.Errorf(
			"domain '%s' expired on %s — cannot create DNS records for expired domains",
			domain.Domain, domain.Expiry.Format("2006-01-02"),
		)
	}

	var id int
	query := `INSERT INTO dns_records (owner, target, domain_id, created_by, updated_by) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = db.QueryRow(query, owner, target, domain.ID, user, user).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to add DNS record: %w", err)
	}

	return LogCreate(db, "dns_record", id, map[string]string{
		"owner":  owner,
		"target": target,
		"domain": domain.Domain,
	}, user)
}

func DNSUpdate(db *sql.DB, owner, newTarget, user string) error {
	domain, err := FindParentDomain(db, owner)
	if err != nil {
		return err
	}

	if IsDomainExpired(domain) {
		return fmt.Errorf(
			"domain '%s' expired on %s — cannot update DNS records for expired domains\n"+
				"  You can still remove them: nurix dns remove --owner %s",
			domain.Domain, domain.Expiry.Format("2006-01-02"), owner,
		)
	}

	// Get current record for changelog
	var currentID int
	var currentTarget string
	err = db.QueryRow(`SELECT id, target FROM dns_records WHERE owner = $1`, owner).Scan(&currentID, &currentTarget)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no DNS record found with owner '%s'", owner)
		}
		return err
	}

	query := `UPDATE dns_records SET target = $1, updated_at = NOW(), updated_by = $2 WHERE owner = $3`
	_, err = db.Exec(query, newTarget, user, owner)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}

	if currentTarget != newTarget {
		LogChange(db, "dns_record", currentID, "UPDATE", "target", currentTarget, newTarget, user)
	}

	return nil
}

func DNSRemove(db *sql.DB, owner, user string) error {
	// Get current record for changelog
	var currentID int
	var currentTarget string
	var currentDomainID int
	err := db.QueryRow(`SELECT id, target, domain_id FROM dns_records WHERE owner = $1`, owner).Scan(&currentID, &currentTarget, &currentDomainID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no DNS record found with owner '%s'", owner)
		}
		return err
	}

	// Log before deleting
	LogDelete(db, "dns_record", currentID, map[string]string{
		"owner":  owner,
		"target": currentTarget,
	}, user)

	_, err = db.Exec(`DELETE FROM dns_records WHERE id = $1`, currentID)
	return err
}

func DNSSearchAll(db *sql.DB, owner, target string) ([]DNSRecord, error) {
	query := `
		SELECT r.id, r.owner, r.target, r.domain_id, d.domain, r.created_by, r.updated_by, r.created_at, r.updated_at
		FROM dns_records r
		JOIN domains d ON r.domain_id = d.id
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if owner != "" {
		query += fmt.Sprintf(" AND r.owner ILIKE $%d", argIdx)
		args = append(args, "%"+owner+"%")
		argIdx++
	}
	if target != "" {
		query += fmt.Sprintf(" AND r.target ILIKE $%d", argIdx)
		args = append(args, "%"+target+"%")
		argIdx++
	}

	query += " ORDER BY d.domain ASC, r.owner ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DNSRecord
	for rows.Next() {
		var r DNSRecord
		if err := rows.Scan(&r.ID, &r.Owner, &r.Target, &r.DomainID, &r.Domain, &r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func GetAllGroupedByDomain(db *sql.DB) ([]Domain, map[int][]DNSRecord, error) {
	domains, err := DomainSearchAll(db, "")
	if err != nil {
		return nil, nil, err
	}

	records, err := DNSSearchAll(db, "", "")
	if err != nil {
		return nil, nil, err
	}

	grouped := make(map[int][]DNSRecord)
	for _, r := range records {
		grouped[r.DomainID] = append(grouped[r.DomainID], r)
	}

	return domains, grouped, nil
}
