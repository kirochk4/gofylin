**Note:** Not done yet.

# Fylin (Fython)

```python
Language = {name = "anon"}

def initLanguage(name):
    return Language{name}

@method(Language, "hello")
def _hello(self):
    println("My name is " + self.name + "!")

fylin = initLanguage("Fylin")
fylin->hello() # My name is Fylin!
```
