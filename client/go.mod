module main

go 1.22

require (
	gopkg.in/yaml.v2 v2.4.0
	util v1.0.0
)

replace (
	util => ../util
)