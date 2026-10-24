package main

import (
	"go/build/constraint"
)

const testHelpersTag = "testhelpers"

// checkBuildTags keeps the testhelpers build tag in the old packages: tests in the target layout keep
// their helpers in test files or in packages that only tests import.
func checkBuildTags(sourceModule *sourceModule) []violation {
	var violations []violation

	for _, file := range sourceModule.files {
		if !isInTargetLayout(file.relativePath) {
			continue
		}

		for _, commentGroup := range file.syntax.Comments {
			if commentGroup.Pos() > file.syntax.Package {
				break
			}

			for _, comment := range commentGroup.List {
				expression, err := constraint.Parse(comment.Text)
				if err != nil || !hasTag(expression, testHelpersTag) {
					continue
				}

				violations = append(violations, violation{
					position: sourceModule.getPosition(file, comment.Pos()),
					rule:     "build-tag",
					message: file.relativePath + " is built only with the " + testHelpersTag +
						" tag; in the target layout test helpers live in test files or in packages only tests import",
				})
			}
		}
	}

	return violations
}

func hasTag(expression constraint.Expr, tag string) bool {
	switch typed := expression.(type) {
	case *constraint.TagExpr:
		return typed.Tag == tag
	case *constraint.NotExpr:
		return hasTag(typed.X, tag)
	case *constraint.AndExpr:
		return hasTag(typed.X, tag) || hasTag(typed.Y, tag)
	case *constraint.OrExpr:
		return hasTag(typed.X, tag) || hasTag(typed.Y, tag)
	default:
		return false
	}
}
