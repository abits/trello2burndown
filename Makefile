.PHONY:
burndown: 
	@curl -w ",\n\n{\"Total time\": \"%{time_total}s\"}\n" -sD - -X POST http://127.0.0.1:8080/burndown --data @payload.json | json --merge | pygmentize -l json
