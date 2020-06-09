package cfg

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type rootStruct struct {
	FirstParam  string
	SecondParam string
	ThirdParam  bool
}

var rootConfig rootStruct

type child2Struct struct {
	FourthParam bool
	FifthParam  int
	SixthParam  string
}

var child2Config child2Struct

var yamlExample = []byte(`
firstparam: First
SecondParam: Second
Nested:
   FourthParam: true
   FifthParam: 78
   SixthParam: Sixth
ThirdParam: false
`)

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	c, err = root.ExecuteC()
	return c, buf.String(), err
}

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(root, args...)
	return output, err
}

func init() {
	ConfigLoader = func() {
		fmt.Println("Test Reading config")
		viper.SetConfigType("yaml")
		err := viper.ReadConfig(bytes.NewBuffer(yamlExample))
		if err != nil {
			fmt.Println(err)
		}
	}
}

func TestRunBoundCommand(t *testing.T) {

	rootCmd := &cobra.Command{
		Use: "root",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("Running root")
		},
	}

	child1Cmd := &cobra.Command{
		Use: "child1",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("Running child1")
		},
	}

	child2Cmd := &cobra.Command{
		Use: "child2",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("Running child2")
		},
	}

	rootCmd.AddCommand(child1Cmd)
	child1Cmd.AddCommand(child2Cmd)

	BindCobraPersistentFlags(rootCmd, &rootConfig)
	BindCobraPersistentFlagsKey("nested", child2Cmd, &child2Config)

	output, err := executeCommand(rootCmd, "child1", "child2", "--fifth-param", "102", "--second-param", "SecondFlag")

	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rootTest := rootStruct{
		FirstParam:  "First",
		SecondParam: "SecondFlag",
		ThirdParam:  false,
	}

	child2Test := child2Struct{
		FourthParam: true,
		FifthParam:  102,
		SixthParam:  "Sixth",
	}

	if rootConfig != rootTest {
		t.Errorf("\ngot:  %v\nwant: %v\n", rootConfig, rootTest)
	}

	if child2Config != child2Test {
		t.Errorf("\ngot:  %v\nwant: %v\n", child2Config, child2Test)
	}
}
