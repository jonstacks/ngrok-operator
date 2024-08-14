package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var commonPredicateFilters = predicate.Or(
	predicate.AnnotationChangedPredicate{},
	predicate.GenerationChangedPredicate{},
)
