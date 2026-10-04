# Inserted into the vendor post-install template only after its successful exit status.
set -eu
user='__GUEST_USER__'
attempt='__ATTEMPT__'
passwd --lock root >/dev/null
test "$(passwd --status root | awk '{print $2}')" = L
id -nG "$user" | tr ' ' '\n' | grep -qx sudo
sudo -l -U "$user" >/dev/null
install -d -o root -g root -m 0755 /var/lib/desky-vm
umask 022
record=$(mktemp /var/lib/desky-vm/installed.XXXXXX)
printf '{"attempt":"%s","user":"%s","root_locked":true,"sudo":true}\n' "$attempt" "$user" > "$record"
chmod 0644 "$record"
chown root:root "$record"
mv -T "$record" /var/lib/desky-vm/installed.json
