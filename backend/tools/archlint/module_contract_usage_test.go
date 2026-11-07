package main

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contractRejection struct {
	name string
	line int
}

func TestModuleContractConsumers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		path        string
		packageName string
		isExternal  bool
	}{
		{
			name: "accepts declarations when application composition is the only consumer",
			path: "internal/app/app.go", packageName: "app", isExternal: true,
		},
		{
			name: "accepts declarations when a command is the only consumer",
			path: "cmd/consumer/main.go", packageName: "main", isExternal: true,
		},
		{
			name: "accepts declarations when external test support is the only consumer",
			path: "internal/app/apptest/app.go", packageName: "apptest", isExternal: true,
		},
		{
			name: "accepts declarations when an external test outside the entry point graph is the only consumer",
			path: "e2e/consumer/consumer_test.go", packageName: "consumer_test", isExternal: true,
		},
		{
			name: "accepts declarations when a similarly named module is the only consumer",
			path: "internal/modules/usersettings/module.go", packageName: "usersettings", isExternal: true,
		},
		{
			name: "rejects declarations when only the owning implementation uses them",
			path: "internal/modules/users/internal/profile/profile.go", packageName: "profile",
		},
		{
			name: "rejects declarations when only a test in the owning implementation uses them",
			path: "internal/modules/users/internal/profile/profile_test.go", packageName: "profile_test",
		},
		{
			name: "rejects declarations when only a separate test package in the owning root uses them",
			path: "internal/modules/users/module_test.go", packageName: "users_test",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			violations := checkFixture(t, map[string]string{
				"internal/modules/users/module.go": "package users\n\ntype Alias = string\nconst Value = 1\n",
				testCase.path: "package " + testCase.packageName + "\n" +
					"import \"fixture/internal/modules/users\"\n" +
					"func Use() { var name users.Alias; _ = name; _ = users.Value }\n",
			})

			if testCase.isExternal {
				require.Empty(t, violations)

				return
			}

			require.Len(t, violations, 2, "violations: %v", violations)
			assert.Equal(t, "internal/modules/users/module.go:3", violations[0].position)
			assert.Equal(t, "internal/modules/users/module.go:4", violations[1].position)
			for _, reported := range violations {
				assert.Equal(t, "module-contract-usage", reported.rule)
			}
		})
	}
}

func TestModuleContractUsage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		contract string
		consumer string
		files    map[string]string
		rejected []contractRejection
	}{
		{
			name:     "rejects an alias when no consumer references it",
			contract: "type Alias = string\n",
			rejected: []contractRejection{{name: "Alias", line: 3}},
		},
		{
			name:     "rejects a constant when no consumer references it",
			contract: "const Value = 1\n",
			rejected: []contractRejection{{name: "Value", line: 3}},
		},
		{
			name:     "accepts declarations when another module references them through a renamed import",
			contract: "type Alias = string\nconst Value = 1\n",
			files: map[string]string{
				"internal/modules/storages/module.go": "package storages\n" +
					"import people \"fixture/internal/modules/users\"\n" +
					"type Deps struct { Name people.Alias }\nconst Size = people.Value\n",
				"internal/app/app.go": "package app\nimport \"fixture/internal/modules/storages\"\n" +
					"func New() { _ = storages.Size }\n",
			},
		},
		{
			name:     "rejects an alias when only comments strings and a shadowed name match",
			contract: "type Alias = string\n",
			files: map[string]string{
				"internal/app/app.go": "package app\n// users.Alias\n" +
					"func New() { Alias := \"users.Alias\"; _ = Alias }\n",
			},
			rejected: []contractRejection{{name: "Alias", line: 3}},
		},
		{
			name:     "ignores defined types functions variables and private declarations when they have no consumer",
			contract: "type Defined string\nfunc New() {}\nvar Value = 1\ntype alias = string\nconst value = 1\n",
		},
		{
			name: "accepts a callback alias when a consumed interface requires it",
			contract: "type Handler = func(string)\n" +
				"type Module interface { Subscribe(Handler) }\n",
			consumer: "var _ users.Module",
		},
		{
			name: "accepts indirect aliases when a consumed type requires them",
			contract: "type Alias = string\ntype Indirect = Alias\n" +
				"type Profile struct { Name Indirect }\n",
			consumer: "var _ users.Profile",
		},
		{
			name: "accepts an embedded alias when a consumed interface requires it",
			contract: "type Embedded = interface { Handle() }\n" +
				"type Module interface { Embedded }\n",
			consumer: "var _ users.Module",
		},
		{
			name: "accepts a constraint alias when a consumed generic signature requires it",
			contract: "type Constraint = interface { ~int }\n" +
				"func Identity[T Constraint](value T) T { return value }\n",
			consumer: "_ = users.Identity(1)",
		},
		{
			name:     "accepts an alias when a consumed constructor returns it",
			contract: "type Alias = string\nfunc New() Alias { return \"\" }\n",
			consumer: "_ = users.New()",
		},
		{
			name: "accepts an alias when a consumed method signature requires it",
			contract: "type Alias = string\ntype Profile struct {}\n" +
				"func (p Profile) GetName() Alias { return \"\" }\n",
			consumer: "_ = users.Profile{}.GetName()",
		},
		{
			name:     "accepts a constant when a consumed type uses it as an array length",
			contract: "const Size = 1\ntype Profile struct { Values [Size]int }\n",
			consumer: "var _ users.Profile",
		},
		{
			name:     "accepts a constant when a consumed variable initializer requires it",
			contract: "const Size = 1\nvar Values = [Size]int{}\n",
			consumer: "_ = users.Values",
		},
		{
			name: "accepts an alias when a consumed function literal signature requires it",
			contract: "type Alias = string\n" +
				"var Handle = func(value Alias) {}\n",
			consumer: "users.Handle(\"\")",
		},
		{
			name:     "rejects aliases when an unused local chain is their only consumer",
			contract: "type Alias = Indirect\ntype Indirect = string\n",
			rejected: []contractRejection{{name: "Alias", line: 3}, {name: "Indirect", line: 4}},
		},
		{
			name:     "rejects an alias when an unused pointer mediated cycle is its only consumer",
			contract: "type Alias = Node\ntype Node struct { Next *Alias }\n",
			rejected: []contractRejection{{name: "Alias", line: 3}},
		},
		{
			name:     "rejects a callback alias when only an unused interface requires it",
			contract: "type Handler = func()\ntype Module interface { Subscribe(Handler) }\n",
			rejected: []contractRejection{{name: "Handler", line: 3}},
		},
		{
			name: "rejects an alias when only a consumed function body uses it",
			contract: "type Alias = string\n" +
				"func New() string { var value Alias; return value }\n",
			consumer: "_ = users.New()",
			rejected: []contractRejection{{name: "Alias", line: 3}},
		},
		{
			name: "rejects an alias when only a consumed function literal body uses it",
			contract: "type Alias = string\n" +
				"var Create = func() string { var value Alias; return value }\n",
			consumer: "_ = users.Create()",
			rejected: []contractRejection{{name: "Alias", line: 3}},
		},
		{
			name:     "rejects unused siblings when only one grouped constant has a consumer",
			contract: "const (\n First = iota\n Second\n Third\n)\n",
			consumer: "_ = users.Second",
			rejected: []contractRejection{{name: "First", line: 4}, {name: "Third", line: 6}},
		},
		{
			name:     "follows implicit expressions when a consumed constant repeats its predecessor",
			contract: "const (\n Base = 1\n Derived = Base + 1\n Repeated\n)\n",
			consumer: "_ = users.Repeated",
			rejected: []contractRejection{{name: "Derived", line: 5}},
		},
		{
			name: "keeps initializer dependencies separate when a constant spec declares several names",
			contract: "const LeftSource = 1\nconst RightSource = 2\n" +
				"const Left, Right = LeftSource, RightSource\n",
			consumer: "_ = users.Left",
			rejected: []contractRejection{{name: "RightSource", line: 4}, {name: "Right", line: 5}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{
				"internal/modules/users/module.go": "package users\n\n" + testCase.contract,
			}
			maps.Copy(files, testCase.files)
			if testCase.consumer != "" {
				files["internal/app/app.go"] = "package app\nimport \"fixture/internal/modules/users\"\n" +
					"func Use() { " + testCase.consumer + " }\n"
			}

			violations := checkFixture(t, files)

			require.Len(t, violations, len(testCase.rejected), "violations: %v", violations)
			for index, rejected := range testCase.rejected {
				assert.Equal(
					t,
					fmt.Sprintf("internal/modules/users/module.go:%d", rejected.line),
					violations[index].position,
				)
				assert.Equal(t, "module-contract-usage", violations[index].rule)
				assert.Contains(t, violations[index].message, rejected.name)
				assert.Contains(t, violations[index].message, "consumer")
			}
		})
	}
}

func TestModuleContractAnalysis(t *testing.T) {
	t.Parallel()

	t.Run("fails analysis when an external consumer cannot resolve its import", func(t *testing.T) {
		t.Parallel()

		root := writeFixture(t, map[string]string{
			"internal/modules/users/module.go": "package users\n\ntype Alias = string\n",
			"internal/app/app.go":              "package app\nimport _ \"fixture/missing\"\n",
		})

		violations, err := check(root, getFixtureSpec())

		require.ErrorIs(t, err, errPackagesHaveErrors)
		assert.Contains(t, err.Error(), "fixture/missing")
		assert.Empty(t, violations)
	})

	t.Run("fails analysis when a contract type cannot be resolved", func(t *testing.T) {
		t.Parallel()

		root := writeFixture(t, map[string]string{
			"internal/modules/users/module.go": "package users\n\ntype Alias = Missing\n",
		})

		violations, err := check(root, getFixtureSpec())

		require.ErrorIs(t, err, errPackagesHaveErrors)
		assert.Contains(t, err.Error(), "Missing")
		assert.Empty(t, violations)
	})

	t.Run("reports each declaration once in stable order with production and test variants", func(t *testing.T) {
		t.Parallel()

		root := writeFixture(t, map[string]string{
			"internal/modules/users/module.go": "package users\n\ntype Alias = string\nconst Value = 1\n",
			"internal/modules/users/module_test.go": "package users\n" +
				"func Use() { var value Alias; _ = value }\n",
			"internal/modules/users/external_test.go": "package users_test\n" +
				"import \"fixture/internal/modules/users\"\nfunc Use() { _ = users.Value }\n",
		})

		firstRun, err := check(root, getFixtureSpec())
		require.NoError(t, err)
		secondRun, err := check(root, getFixtureSpec())
		require.NoError(t, err)

		require.Len(t, firstRun, 2)
		assert.Equal(t, firstRun, secondRun)
		assert.Equal(t, "internal/modules/users/module.go:3", firstRun[0].position)
		assert.Equal(t, "internal/modules/users/module.go:4", firstRun[1].position)
		assert.Contains(t, firstRun[0].message, "Alias")
		assert.Contains(t, firstRun[1].message, "Value")
		for _, reported := range firstRun {
			assert.Equal(t, "module-contract-usage", reported.rule)
		}
	})

	t.Run("ignores an inactive contract when the default build excludes it", func(t *testing.T) {
		t.Parallel()

		violations := checkFixture(t, map[string]string{
			"internal/modules/users/module.go": "//go:build archlint_inactive\n\npackage users\n\ntype Alias = string\n",
			"internal/app/app.go":              "package app\n",
		})

		require.Empty(t, violations)
	})

	t.Run("ignores an inactive contract when another contract triggers type analysis", func(t *testing.T) {
		t.Parallel()

		violations := checkFixture(t, map[string]string{
			"internal/modules/users/module.go":    "//go:build archlint_inactive\n\npackage users\n\ntype Alias = string\n",
			"internal/modules/storages/module.go": "package storages\nconst Value = 1\n",
			"internal/app/app.go": "package app\nimport \"fixture/internal/modules/storages\"\n" +
				"func Use() { _ = storages.Value }\n",
		})

		require.Empty(t, violations)
	})
}
