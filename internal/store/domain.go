package store

import (
	"database/sql"
	"fmt"
	"time"
)

type Domain struct {
	ID        int
	Domain    string
	Provider  string
	Expiry    time.Time
	CreatedBy string
	UpdatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func DomainAdd(db *sql.DB, domain, provider string, expiry time.Time, user string) error {
	var id int
	query := `INSERT INTO domains (domain, provider, expiry, created_by, updated_by) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := db.QueryRow(query, domain, provider, expiry, user, user).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}

	// Log to changelog
	return LogCreate(db, "domain", id, map[string]string{
		"domain":   domain,
		"provider": provider,
		"expiry":   expiry.Format("2006-01-02"),
	}, user)
}

func DomainUpdate(db *sql.DB, domain string, provider *string, expiry *time.Time, user string) error {
	// Fetch current values for changelog
	current, err := GetDomainByName(db, domain)
	if err != nil {
		return err
	}

	setClauses := "updated_at = NOW(), updated_by = $1"
	args := []interface{}{user}
	argIdx := 2

	if provider != nil {
		setClauses += fmt.Sprintf(", provider = $%d", argIdx)
		args = append(args, *provider)
		argIdx++
	}
	if expiry != nil {
		setClauses += fmt.Sprintf(", expiry = $%d", argIdx)
		args = append(args, *expiry)
		argIdx++
	}

	args = append(args, domain)
	query := fmt.Sprintf("UPDATE domains SET %s WHERE domain = $%d", setClauses, argIdx)

	res, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update domain: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no domain found: '%s'", domain)
	}

	// Log changes
	if provider != nil && *provider != current.Provider {
		LogChange(db, "domain", current.ID, "UPDATE", "provider", current.Provider, *provider, user)
	}
	if expiry != nil && !expiry.Equal(current.Expiry) {
		LogChange(db, "domain", current.ID, "UPDATE", "expiry", current.Expiry.Format("2006-01-02"), expiry.Format("2006-01-02"), user)
	}

	return nil
}

func DomainRemove(db *sql.DB, domain string, user string) error {
	current, err := GetDomainByName(db, domain)
	if err != nil {
		return err
	}

	// Check for existing DNS records
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dns_records WHERE domain_id = $1`, current.ID).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf(
			"cannot remove domain '%s' — it still has %d DNS record(s)\n\n"+
				"  List them:   nurix dns search all --owner %s\n"+
				"  Remove them: nurix dns remove --owner <record>\n\n"+
				"Clean up all DNS records first, then retry.",
			domain, count, domain,
		)
	}

	// Log before deleting
	LogDelete(db, "domain", current.ID, map[string]string{
		"domain":   current.Domain,
		"provider": current.Provider,
		"expiry":   current.Expiry.Format("2006-01-02"),
	}, user)

	_, err = db.Exec(`DELETE FROM domains WHERE id = $1`, current.ID)
	return err
}

func DomainSearchAll(db *sql.DB, domainFilter string) ([]Domain, error) {
	query := `SELECT id, domain, provider, expiry, created_by, updated_by, created_at, updated_at FROM domains WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if domainFilter != "" {
		query += fmt.Sprintf(" AND domain ILIKE $%d", argIdx)
		args = append(args, "%"+domainFilter+"%")
		argIdx++
	}

	query += " ORDER BY domain ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.Domain, &d.Provider, &d.Expiry, &d.CreatedBy, &d.UpdatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func GetDomainByName(db *sql.DB, domain string) (*Domain, error) {
	var d Domain
	err := db.QueryRow(
		`SELECT id, domain, provider, expiry, created_by, updated_by, created_at, updated_at FROM domains WHERE domain = $1`,
		domain,
	).Scan(&d.ID, &d.Domain, &d.Provider, &d.Expiry, &d.CreatedBy, &d.UpdatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain '%s' is not registered in nurix", domain)
		}
		return nil, err
	}
	return &d, nil
}

// FindParentDomain finds the registered base domain for a given hostname.
func FindParentDomain(db *sql.DB, owner string) (*Domain, error) {
	// Try the owner itself first
	d, err := GetDomainByName(db, owner)
	if err == nil {
		return d, nil
	}

	// Try progressively shorter suffixes
	parts := splitDomain(owner)
	for i := 1; i < len(parts); i++ {
		candidate := joinDomain(parts[i:])
		d, err := GetDomainByName(db, candidate)
		if err == nil {
			return d, nil
		}
	}

	return nil, fmt.Errorf(
		"no registered domain found for '%s'\n\n"+
			"  Register the base domain first:\n"+
			"  nurix domain add --domain <base-domain> --provider <provider> --expiry <YYYY-MM-DD>",
		owner,
	)
}

func IsDomainExpired(d *Domain) bool {
	today := time.Now().Truncate(24 * time.Hour)
	expiry := d.Expiry.Truncate(24 * time.Hour)
	return today.After(expiry)
}

func splitDomain(domain string) []string {
	var parts []string
	current := ""
	for _, ch := range domain {
		if ch == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func joinDomain(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "."
		}
		result += p
	}
	return result
}
