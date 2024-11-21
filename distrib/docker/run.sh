#!/bin/sh

# make config file
if [ ! -f "/ircd/ircd.yaml" ]; then
    awk '{gsub(/path: languages/,"path: /ircd-bin/languages")}1' /ircd-bin/default.yaml > /tmp/ircd.yaml

    # change default oper passwd
    OPERPASS=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c20)
    echo "Oper username:password is admin:$OPERPASS"
    ENCRYPTEDPASS=$(echo "$OPERPASS" | /ircd-bin/ergo genpasswd)
    ORIGINALPASS='\$2a\$04\$0123456789abcdef0123456789abcdef0123456789abcdef01234'

    awk "{gsub(/password: \\\"$ORIGINALPASS\\\"/,\"password: \\\"$ENCRYPTEDPASS\\\"\")}1" /tmp/ircd.yaml > /tmp/ircd2.yaml

    unset OPERPASS
    unset ENCRYPTEDPASS
    unset ORIGINALPASS

    mv /tmp/ircd2.yaml /ircd/ircd.yaml
fi

# make self-signed certs if they don't already exist
/ircd-bin/ergo mkcerts

# run!
exec /ircd-bin/ergo run