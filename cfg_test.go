package cfg

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var yamlExample = []byte(`
firstparam: First
SecondParam: Second
Nested:
   FourthParam: true
   FifthParam: 78
   SixthParam: Sixth
ThirdParam: false
CollectionSelectedItem: SecondItem
collection:
 - seventhParam: false
   eighthParam: FirstEighth
   name: FirstItem
 - seventhParam: true
   eighthParam: SecondEighth
   name: SecondItem
 - seventhParam: false
   eighthParam: ThirdEighth
   name: ThirdItem
ninthParam: Ninth
tenthParam: 9
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

type rootStruct struct {
	FirstParam  string
	SecondParam string
	ThirdParam  bool
}

type rootStruct2 struct {
	NinthParam string
	TenthParam int
}

type child2Struct struct {
	FourthParam bool
	FifthParam  int
	SixthParam  string
}

func TestRunBoundCommand(t *testing.T) {

	var (
		rootConfig   rootStruct
		rootConfig2  rootStruct2
		child2Config child2Struct
	)

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

	BindPersistentFlags(rootCmd, &rootConfig)
	BindPersistentFlags(rootCmd, &rootConfig2)
	BindPersistentFlagsKey("nested", child2Cmd, &child2Config)

	output, err := executeCommand(rootCmd, "child1", "child2", "--fifth-param", "102", "--second-param", "SecondFlag", "--tenth-param", "10")

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

	rootTest2 := rootStruct2{
		NinthParam: "Ninth",
		TenthParam: 10,
	}

	child2Test := child2Struct{
		FourthParam: true,
		FifthParam:  102,
		SixthParam:  "Sixth",
	}

	if rootConfig != rootTest {
		t.Errorf("\ngot:  %v\nwant: %v\n", rootConfig, rootTest)
	}

	if rootConfig2 != rootTest2 {
		t.Errorf("\ngot:  %v\nwant: %v\n", rootConfig2, rootTest2)
	}

	if child2Config != child2Test {
		t.Errorf("\ngot:  %v\nwant: %v\n", child2Config, child2Test)
	}
}

type collRootStruct struct {
	FirstParam             string
	SecondParam            string
	ThirdParam             bool
	CollectionSelectedItem string
}

type itemStruct struct {
	SeventhParam bool
	EighthParam  string
	Name         string
}

func TestRunBoundCollectionCommand(t *testing.T) {

	var (
		rootConfig collRootStruct
		itemConfig itemStruct
	)

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

	BindPersistentFlags(rootCmd, &rootConfig)
	BindPersistentFlagsCollection("collection", "CollectionSelectedItem", child2Cmd, &itemConfig)

	output, err := executeCommand(rootCmd, "child1", "child2", "--eighth-param", "SecondEighthFlag", "--second-param", "SecondFlag")

	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rootTest := collRootStruct{
		FirstParam:             "First",
		SecondParam:            "SecondFlag",
		ThirdParam:             false,
		CollectionSelectedItem: "SecondItem",
	}

	itemTest := itemStruct{
		SeventhParam: true,
		EighthParam:  "SecondEighthFlag",
		Name:         "SecondItem",
	}

	if rootConfig != rootTest {
		t.Errorf("\ngot:  %v\nwant: %v\n", rootConfig, rootTest)
	}

	if itemConfig != itemTest {
		t.Errorf("\ngot:  %v\nwant: %v\n", itemConfig, itemTest)
	}
}
