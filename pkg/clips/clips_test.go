package clips

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/udhos/equalfile"
	"gotest.tools/assert"
)

func TestCreateEnvironment(t *testing.T) {
	t.Run("Explicit close", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()
	})

	t.Run("Load text", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Load("testdata/dopey.save")
		assert.NilError(t, err)
	})

	t.Run("Load binary", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Load("testdata/dopey.bsave")
		assert.NilError(t, err)
	})

	t.Run("Load failure", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Load("testdata/file_not_found")
		assert.ErrorContains(t, err, "Unable")
	})

	t.Run("Save text", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Load("testdata/dopey.save")
		assert.NilError(t, err)

		tmpfile, err := ioutil.TempFile("", "test.*.save")
		assert.NilError(t, err)
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		err = env.Save(tmpfile.Name(), false)
		assert.NilError(t, err)

		cmp := equalfile.New(nil, equalfile.Options{})
		equal, err := cmp.CompareFile("testdata/dopey.save", tmpfile.Name())
		assert.NilError(t, err)
		assert.Equal(t, equal, true)
	})

	t.Run("Save binary", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Load("testdata/dopey.save")
		assert.NilError(t, err)

		tmpfile, err := ioutil.TempFile("", "test.*.save")
		assert.NilError(t, err)
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		err = env.Save(tmpfile.Name(), true)
		assert.NilError(t, err)

		// Binary output is not consistent; not sure how to verify
		/*
			cmp := equalfile.New(nil, equalfile.Options{})
			equal, err := cmp.CompareFile("testdata/dopey.bsave", tmpfile.Name())
			assert.NilError(t, err)
			assert.Equal(t, equal, true)
		*/
	})

	t.Run("Save failure", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Load("testdata/dopey.save")
		assert.NilError(t, err)

		err = env.Save("/not_writable", true)
		assert.ErrorContains(t, err, "Unable")
	})

	t.Run("BatchStar", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.BatchStar("testdata/dopey.clp")
		assert.NilError(t, err)
	})

	t.Run("BatchStar failure", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.BatchStar("testdata/file_not_found")
		assert.ErrorContains(t, err, "Unable")
	})

	t.Run("Build", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Build("(deftemplate foo (slot bar))")
		assert.NilError(t, err)
	})

	t.Run("Build failure", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		err := env.Build("(deftemplate foo (slot bar")
		assert.ErrorContains(t, err, "Unable")
	})

	t.Run("Eval", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		ret, err := env.Eval("(rules)")
		assert.NilError(t, err)
		assert.Equal(t, ret, nil)
	})

	t.Run("Eval Failure", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		_, err := env.Eval("(create$ 1 2 3")
		assert.ErrorContains(t, err, "Unable to parse")
	})

	t.Run("Clear", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		env.Clear()
	})

	t.Run("Reset", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		env.Reset()
	})

	t.Run("DefineFunction", func(t *testing.T) {
		env := CreateEnvironment()
		defer env.Close()

		argcount := 0
		callback := func(args []interface{}) (interface{}, error) {
			argcount = len(args)
			return nil, nil
		}

		err := env.DefineFunction("test-callback", callback)
		assert.NilError(t, err)

		_, err = env.Eval("(test-callback a b c)")
		assert.NilError(t, err)
		assert.Equal(t, argcount, 3)
	})
}
