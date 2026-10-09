# Reading Group Site

To deploy:

Create a user and set up the directory structure:

```````
addgroup -S readinggroup
adduser -D -S -H -s /sbin/nologin -G readinggroup -g readinggroup readinggroup
mkdir -p /opt/reading-group/data
chown -R readinggroup:readinggroup /opt/reading-group
chmod 750 /opt/reading-group
chmod 770 /opt/reading-group/data
```````

Fetch the binary from Github:

`````
wget https://github.com/Binkersss/reading-group/releases/download/v0.0.1/reading-group \
    -O /opt/reading-group/reading-group
chown readinggroup:readinggroup /opt/reading-group/reading-group
chmod 750 /opt/reading-group/reading-group
`````

Copy the [OpenRC configuration](./reading-group.conf) to
`/etc/conf.d/reading-group`.

Copy the [OpenRC service](./reading-group.openrc) to `/etc/init.d/reading-group`
and make it executable.

Start and enable the service:

`````
rc-service reading-group start
rc-update add reading-group default
`````
