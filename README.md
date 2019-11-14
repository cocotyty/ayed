# AYED
Another YAML command-line Editor.
## INSTALL
```bash
go get -u github.com/cocotyty/ayed
```
## USAGE
### EDIT EXAMPLE
source.yaml:
```yaml
doc:
  title:
    text: AYED
  content:
    - text: "the reason for use this project"
      color: red
  status:
   create: 2019-02-14
```
script.yaml:
```yaml
commands:
# delete doc.title
  - path: doc\.title
    action: delete
# add field `update` to doc.status
  - has_fields:
      create: .*
    action: merge
    params:
      update: 2019-02-15
# rewrite doc.content.0.text value
  - path: doc\.content\.0\.text
    action: replace
    params: "no good documents"
# append new line to doc.content
  - path: doc\.content
    action: append
    params:
      text: "second row"
      color: blue
# append lines to doc.content
  - path: doc\.content
    action: merge
    params:
      - text: "third row"
        color: yellow
      - text: "final"
        color: yellow
```

execute commands:
```sh
ayed -s script.yaml -f source.yaml
```
output:
```yaml
doc:
  content:
  - text: no good documents
    color: red
  - text: second row
    color: blue
  - text: third row
    color: yellow
  - text: "final"
    color: yellow
  status:
    create: "2019-02-14"
    update: "2019-02-15"
```
### READ EXAMPLE
source.yaml:
```yaml
doc:
  title:
    text: AYED
  content:
    - text: "the reason for use this project"
      color: red
  status:
   create: 2019-02-14
```
#### example 1
```bash
ayed -f ./source.yaml -r 'content.0.text' -p 'doc' 
```
will print:
```text
the reason for use this project
```

#### example 2
```bash
ayed -f ./source.yaml -r text -m color=red 
```
will print:
```text
the reason for use this project
```

#### example 3
```bash
ayed -f ./source.yaml -r text  -p doc.title.text
```
will print:
```text
AYED
```

### script rules:
- **path** is regex that will match the yaml node path you want to edited.
- **has_fields** is a map that will match the yaml map node.
- **action: delete**  delete the matched node.
- **action: merge**  merge params into the matched node.
- **action: replace**  replace the matched node with params.
- **action: append**  append params to the matched array node.
