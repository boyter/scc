// movies.cypher
// Example query to test scc complexity definitions for Cypher.

/*
  This multi-line block explains the query's purpose.
  It finds actors who co-starred with "Tom Hanks" and then
  looks for paths of up to 5 steps to "Kevin Bacon".
*/

// Find Tom Hanks' co-actors
MATCH (tom:Actor {name: 'Tom Hanks'})-[:ACTED_IN]->(m:Movie)<-[:ACTED_IN]-(coActor:Actor)
WHERE coActor.name <> "Tom Hanks"

// Now, check for a connection to Kevin Bacon
WITH coActor
MATCH (bacon:Actor {name: "Kevin Bacon"})
// This is a variable-length path, indicating higher complexity
MATCH p = shortestPath((coActor)-[*..5]-(bacon))

// UNION is another complexity check
UNION

// An optional match for actors born after 1960
MATCH (youngActor:Actor)
WHERE youngActor.born > 1960
OPTIONAL MATCH (youngActor)-[:DIRECTED]->(anyMovie:Movie)

RETURN youngActor.name, anyMovie.title;