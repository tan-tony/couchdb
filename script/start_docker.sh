#!/bin/bash

# This script is meant to run locally while testing Kivik. It starts various
# versions of CouchDB in docker, for testing.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"


export COUCHDB_USER=admin
export COUCHDB_PASSWORD=abc123
export KIVIK_TEST_DSN_COUCH17=http://admin:abc123@localhost:6001/
export KIVIK_TEST_DSN_COUCH22=http://admin:abc123@localhost:6002/
export KIVIK_TEST_DSN_COUCH23=http://admin:abc123@localhost:6003/
export KIVIK_TEST_DSN_COUCH30=http://admin:abc123@localhost:6004/
export KIVIK_TEST_DSN_COUCH31=http://admin:abc123@localhost:6005/

echo "CouchDB 1.7.2"
docker pull couchdb:1.7.2
docker run --name couch17 -p 6001:5984/tcp -d --rm -e COUCHDB_USER -e COUCHDB_PASSWORD couchdb:1.7.2
${DIR}/complete_couch1.sh $KIVIK_TEST_DSN_COUCH17

echo "CouchDB 2.2.0"
docker pull couchdb:2.2.0
docker run --name couch22 -p 6002:5984/tcp -d --rm -e COUCHDB_USER -e COUCHDB_PASSWORD couchdb:2.2.0
${DIR}/complete_couch2.sh $KIVIK_TEST_DSN_COUCH22

echo "CouchDB 2.3.1"
docker pull apache/couchdb:2.3.1
docker run --name couch23 -p 6003:5984/tcp -d --rm -e COUCHDB_USER -e COUCHDB_PASSWORD apache/couchdb:2.3.1
${DIR}/complete_couch2.sh $KIVIK_TEST_DSN_COUCH23

echo "CouchDB 3.0.0"
docker pull couchdb:3.0.0
docker run --name couch30 -p 6004:5984/tcp -d --rm -e COUCHDB_USER -e COUCHDB_PASSWORD apache/couchdb:3.0.0
${DIR}/complete_couch2.sh $KIVIK_TEST_DSN_COUCH30

echo "CouchDB 3.1.1"
docker pull apache/couchdb:3.1.1
docker run --name couch31 -p 6005:5984/tcp -d --rm -e COUCHDB_USER -e COUCHDB_PASSWORD apache/couchdb:3.1.1
${DIR}/complete_couch2.sh $KIVIK_TEST_DSN_COUCH31
