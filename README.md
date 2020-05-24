# cfg

Convenience package for abstracting away Viper configurations and Cobra flags.

It takes a Viper configuration file like:

```yaml
root:
    FirstVar: 'Value1'
    SecondVar: 'Value2'
    ThirdVar: 'Value3'
```

Cobra flags like:

```
app --first-var Override1 --third-var Override3
```

And resolves them into a struct with initial values:

```go
package cmd

import (
    "fmt"

    "github.com/bartdeboer/cfg"
    "github.com/spf13/cobra"
)

type Config struct {
    FirstVar  string `usage:"First var description"`
    SecondVar string `usage:"Second var description"`
    ThirdVar  string `usage:"Third var description"`
}

var initial Config

var rootCmd = &cobra.Command{
    Use:   "root",
    Short: "A brief description of your application",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println(initial.FirstVar) // returns Override1
        fmt.Println(initial.SecondVar) // returns Value2
        fmt.Println(initial.ThirdVar) // returns Override3
    },
}

func init() {
    cfg.BindPersistentFlagsKey("root", rootCmd, &initial)
}
```

Where Cobra flags are overrides for Viper values
