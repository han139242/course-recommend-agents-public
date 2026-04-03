#!/bin/bash
set -euo pipefail

NEO4J_URI="${NEO4J_URI:-bolt://neo4j:7687}"
NEO4J_USER="${NEO4J_USER:-neo4j}"
NEO4J_PASSWORD="${NEO4J_PASSWORD:-please-change-me}"
NEO4J_DATABASE="${NEO4J_DATABASE:-neo4j}"
SEED_FILE="${SEED_FILE:-/seed/knowledge_graph_seed.cypher}"

echo "Waiting for Neo4j at ${NEO4J_URI}..."
for i in $(seq 1 60); do
  if cypher-shell -a "${NEO4J_URI}" -u "${NEO4J_USER}" -p "${NEO4J_PASSWORD}" "RETURN 1;" >/dev/null 2>&1; then
    echo "Neo4j is ready."
    break
  fi
  sleep 2
done

if [ ! -f "${SEED_FILE}" ]; then
  echo "Seed file not found: ${SEED_FILE}"
  exit 1
fi

echo "Importing seed graph from ${SEED_FILE}..."
cypher-shell -a "${NEO4J_URI}" -u "${NEO4J_USER}" -p "${NEO4J_PASSWORD}" -d "${NEO4J_DATABASE}" -f "${SEED_FILE}"
echo "Neo4j seed import completed."
