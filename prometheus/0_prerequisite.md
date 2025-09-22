docker exec -it clickhouse-server ls /etc/clickhouse-server/users.d/

docker exec -it clickhouse-server clickhouse-client --query "SELECT name FROM system.users;"

Create a new user config file
Create a file otel-user.xml inside users.d:
docker exec -it clickhouse-server bash
cd /etc/clickhouse-server/users.d/
nano otel-user.xml
Add this content:
<clickhouse>
    <users>
        <otel_user>
            <password>otel_pass</password>
            <networks>
                <ip>::/0</ip>
            </networks>
            <profile>default</profile>
            <quota>default</quota>
        </otel_user>
    </users>
</clickhouse>
Save and exit.
Restart ClickHouse
docker restart clickhouse-server
Test authentication
curl -u otel_user:otel_pass http://localhost:9363/read