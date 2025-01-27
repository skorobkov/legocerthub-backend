package sqlite

import (
	"database/sql"
	"legocerthub-backend/pkg/acme"
	"legocerthub-backend/pkg/domain/orders"
	"legocerthub-backend/pkg/domain/private_keys"
)

// orderDb is a single acme order, as database table fields
// corresponds to orders.Order
type orderDb struct {
	id             int
	certificate    certificateDb
	location       string
	status         string
	knownRevoked   bool
	err            sql.NullString // stored as json object
	expires        sql.NullInt32
	dnsIdentifiers commaJoinedStrings // will be a comma separated list from storage
	authorizations commaJoinedStrings // will be a comma separated list from storage
	finalize       string
	finalizedKey   keyDb
	certificateUrl sql.NullString
	pem            sql.NullString
	validFrom      sql.NullInt32
	validTo        sql.NullInt32
	createdAt      int
	updatedAt      int
}

func (order orderDb) toOrder(store *Storage) orders.Order {
	// handle if key is not null (id value would not be okay from coalesce if null)
	var key *private_keys.Key
	if order.finalizedKey.id >= 0 {
		key = new(private_keys.Key)
		*key = order.finalizedKey.toKey()
	}

	// handle acme Error
	var acmeErr *acme.Error
	if order.err.Valid {
		acmeErr = acme.NewAcmeError(&order.err.String)
	}

	return orders.Order{
		ID:             order.id,
		Certificate:    order.certificate.toCertificate(store),
		Location:       order.location,
		Status:         order.status,
		KnownRevoked:   order.knownRevoked,
		Error:          acmeErr,
		Expires:        nullInt32ToInt(order.expires),
		DnsIdentifiers: order.dnsIdentifiers.toSlice(),
		Authorizations: order.authorizations.toSlice(),
		Finalize:       order.finalize,
		FinalizedKey:   key,
		CertificateUrl: nullStringToString(order.certificateUrl),
		Pem:            nullStringToString(order.pem),
		ValidFrom:      nullInt32ToInt(order.validFrom),
		ValidTo:        nullInt32ToInt(order.validTo),
		CreatedAt:      order.createdAt,
		UpdatedAt:      order.updatedAt,
	}
}
