#!/bin/sh

if [ "$#" -ne 1 ]; then
    exit 1
fi

if [ "$1" = "_assetInfo" ]; then
    printf '%s\n' '{ "type": "conjure-ir-extensions-provider" }'
    exit 0
fi

printf '%s\n' '{}'
exit 0