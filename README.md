# Reading Group Site

To deploy (assuming alpine linux):

Create a user and set up the directory structure:

```````
addgroup -S readinggroup
adduser -D -S -H -s /sbin/nologin -G readinggroup -g readinggroup readinggroup
mkdir -p /opt/reading-group/data
mkdir /opt/reading-group/static
chown -R readinggroup:readinggroup /opt/reading-group
chown -R readinggroup:readinggroup /opt/reading-group/static
chmod 750 /opt/reading-group
chmod 770 /opt/reading-group/data
```````

Fetch the binary from Github:

`````
wget https://github.com/Binkersss/reading-group/releases/download/v0.0.3/reading-group \
    -O /opt/reading-group/reading-group
wget https://raw.githubusercontent.com/Binkersss/reading-group/master/static/styles.css -O /opt/reading-group/static/styles.css
chown readinggroup:readinggroup /opt/reading-group/reading-group
chown readinggroup:readinggroup /opt/reading-group/static/styles.css
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
