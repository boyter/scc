// A singleline comment
/*
  A multiline comment
  /*
    Nested multiline
  */
*/

/proc/foo()
	for(var/turf/station/T in world)
		if(T.name == "floor")
			del(T)
		T.color = "red"

/turf/station
	icon = 'station.dmi'
	icon_state = "wall"
	var/description = {"
		Multiline
		string
	"}
