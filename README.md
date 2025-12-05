**Note:** Not done yet.

# Fylin (Fython)

```python
Language = { name: "Anon" }

def init_Language(name):
    return Language{ name }

@method(Language, "hello")
def _hello(self):
    println("My name is " + self.name + "!")

fylin = init_Language("Fylin")
fylin->hello() # My name is Fylin!
```
