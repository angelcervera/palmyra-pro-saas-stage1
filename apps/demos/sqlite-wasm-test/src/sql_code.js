export function execSQL(db) {
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
