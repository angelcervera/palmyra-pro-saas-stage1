import sqlite3InitModule from '@sqlite.org/sqlite-wasm';

const log = console.log;
const error = console.error;

const start = (sqlite3) => {
    log('Running SQLite3 version', sqlite3.version.libVersion);
    let db;
    if ('opfs' in sqlite3) {
        db = new sqlite3.oo1.OpfsDb('/mydb.sqlite3');
        log('OPFS is available, created persisted database at', db.filename);
    } else {
        db = new sqlite3.oo1.DB('/mydb.sqlite3', 'ct');
        log('OPFS is not available, created transient database', db.filename);
    }

    // Your SQLite code here.
    execSQL(db);
};

const initializeSQLite = async () => {
    try {
        log('Loading and initializing SQLite3 module...');
        const sqlite3 = await sqlite3InitModule({
            print: log,
            printErr: error,
        });
        log('Done initializing. Running demo...');
        start(sqlite3);
    } catch (err) {
        error('Initialization error:', err.name, err.message);
    }
};

function execSQL(db) {
    try {
        log('Creating a table...');
        db.exec('CREATE TABLE IF NOT EXISTS t(a,b)');
        log('Insert some data using exec()...');
        for (let i = 20; i <= 25; ++i) {
            db.exec({
                sql: 'INSERT INTO t(a,b) VALUES (?,?)',
                bind: [i, i * 2],
            });
        }
        log('Query data with exec()...');
        db.exec({
            sql: 'SELECT a FROM t ORDER BY a LIMIT 3',
            callback: (row) => {
                log(row);
            },
        });
    } finally {
        db.close();
    }
}

initializeSQLite();
