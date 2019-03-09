# Logentries Hook for [Logrus](https://github.com/sirupsen/logrus)

Package logentriesrus provides a logentries hook for the [logrus] logging package.

## Example usage

```
package main

import (
    "os"

    "github.com/sirupsen/logrus"
    "github.com/jcftang/logentriesrus"
)

func main() {

    logrus.SetFormatter(&logrus.JSONFormatter{})

    logrus.SetOutput(os.Stderr)

    logrus.SetLevel(logrus.DebugLevel)

    le, err := logentriesrus.NewLogentriesrusHook("5605593B-9E4F-4A3E-9865-12752055E14B")
    if err != nil {
        os.Exit(-1)        
    }
    logrus.AddHook(le)

    logrus.WithFields(logrus.Fields{"foo": "bar", "foo2": 42}).Warn("this is a warn level message")
    logrus.Info("this is an info level message")
    logrus.Debug("this is a debug level message")
}
```

[logrus]: https://github.com/sirupsen/logrus
