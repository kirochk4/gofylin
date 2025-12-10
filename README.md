**Note:** Not done yet.

# Yeva

```python
Language = { name: "Anon" }

def init_Language(name):
    return Language{ name }

@method(Language, "hello")
def _hello(self):
    println("My name is " + self.name + "!")

yeva = init_Language("Yeva")
yeva->hello() # My name is Yeva!
```
