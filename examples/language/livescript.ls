# Easy listing of implicit objects
table1 =
  * id: 1
    name: 'george'
  * id: 2
    name: 'mike'
  * id: 3
    name: 'donald'





table2 =
  * id: 2
    age: 21
  * id: 1
    age: 20
  * id: 3
    age: 26




# Implicit access, accessignment
up-case-name = (.name .= to-upper-case!)

# List comprehensions, destructuring, piping
[{id:id1, name, age} for {id:id1, name} in table1
                     for {id:id2, age} in table2
                     when id1 is id2]
|> sort-by (.id) # using 'sort-by' from prelude.ls
|> each up-case-name # using 'each' from prelude.ls
|> JSON.stringify
#=>
#[{"id":1,"name":"GEORGE","age":20},
# {"id":2,"name":"MIKE",  "age":21},
# {"id":3,"name":"DONALD","age":26}]











# operators as functions, piping
map (.age), table2 |> fold1 (+)
#=> 67 ('fold1' and 'map' from prelude.ls)