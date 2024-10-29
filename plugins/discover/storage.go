// Copyright 2018-present the CoreDHCP Authors. All rights reserved
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package discover

import (
	"database/sql"
	"errors"
	"fmt"
	"net"

	_ "github.com/mattn/go-sqlite3"
)

func loadDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, fmt.Errorf("failed to open database (%T): %w", err, err)
	}
	if _, err := db.Exec("create table if not exists server4 (mac string not null, state string not null, bootfile string not null, ip string not null, label string not null, primary key (mac, state))"); err != nil {
		return nil, fmt.Errorf("table creation failed: %w", err)
	}
	return db, nil
}

// loadRecords loads the State Records global map with records stored on
// the specified file. The records have to be one per line, a mac address and a state
func loadRecords(db *sql.DB) (map[string]*Record, error) {
	rows, err := db.Query("select mac, state, bootfile, ip, label from server4")
	if err != nil {
		return nil, fmt.Errorf("failed to query server database: %w", err)
	}
	defer rows.Close()
	var (
		mac, state, bootfile, ip, label string
		records                  = make(map[string]*Record)
	)
	for rows.Next() {
		if err := rows.Scan(&mac, &state, &bootfile, &ip, &label); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		hwaddr, err := net.ParseMAC(mac)
		if err != nil {
			return nil, fmt.Errorf("malformed hardware address: %s", mac)
		}
		records[hwaddr.String()] = &Record{state: state, bootfile: bootfile, ip: ip, label: label}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed server database row scanning: %w", err)
	}
	return records, nil
}

func (p *PluginState) saveServer(mac net.HardwareAddr, record *Record) error {
	stmt, err := p.serverdb.Prepare(`insert or replace into server4(mac, state, bootfile, ip, label) values (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("statement preparation failed: %w", err)
	}
	if _, err := stmt.Exec(
		mac.String(),
		record.state,
		record.bootfile,
		record.ip,
		record.label,
	); err != nil {
		return fmt.Errorf("record insert/update failed: %w", err)
	}
	return nil
}

func (p *PluginState) deleteServer(mac string) error {
	stmt, err := p.serverdb.Prepare(`delete from server4 where (mac) = (?)`)
	if err != nil {
		return fmt.Errorf("statement preparation failed: %w", err)
	}
	if _, err := stmt.Exec(
		mac,
	); err != nil {
		return fmt.Errorf("record delete failed: %w", err)
	}
	return nil
}

// registerBackingDB installs a database connection string as the backing store for leases
func (p *PluginState) registerBackingDB(filename string) error {
	if p.serverdb != nil {
		return errors.New("cannot swap out a lease database while running")
	}
	// We never close this, but that's ok because plugins are never stopped/unregistered
	newServerDB, err := loadDB(filename)
	if err != nil {
		return fmt.Errorf("failed to open lease database %s: %w", filename, err)
	}
	p.serverdb = newServerDB
	return nil
}
