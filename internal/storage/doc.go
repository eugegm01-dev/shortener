// Package storage provides implementations of URL storage backends.
//
// It defines the Storage interface and supplies three concrete
// implementations: in‑memory (MemoryStorage), file‑based (FileStorage),
// and PostgreSQL (PostgresStorage). All storages support user‑scoped
// operations, batch saving, and soft deletion.
package storage
