root:root@tcp(localhost:3306)/appdb?parseTime=true

export MYSQL_DSN=root:root@tcp(localhost:3306)/appdb?parseTime=true


data path:
/Users/dkgosql/tmp/invoice-data
file name: invoice-{orderid}.pdf

mysqldump -u root -proot#123PD dkgosql_cloud_dbs > dkgosql_cloud_dbs.sql

mysql -u root -proot#123PD dkgosql_cloud_dbs < dkgosql_cloud_dbs.sql

