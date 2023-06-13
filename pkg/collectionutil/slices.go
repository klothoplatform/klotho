package collectionutil

// FlattenUnique appends each unique item in each list, in the order in which it first appears.
//
// Examples:
//   - FlattenUnique([]int{1, 2, 3}, []int{4, 3, 4}) => []int{1, 2, 3, 4}
//   - FlattenUnique([]int{1, 2, 2}, []int{3, 4}) => []int{1, 2, 3, 4}
func FlattenUnique[E comparable](slices ...[]E) []E {
	alreadyInResult := make(map[E]struct{})

	var result []E
	for _, slice := range slices {
		for _, elem := range slice {
			_, alreadyIn := alreadyInResult[elem]
			if !alreadyIn {
				result = append(result, elem)
				alreadyInResult[elem] = struct{}{}
			}
		}

	}

	return result
}

func Contains[E comparable](slice []E, elem E) bool {
	for _, e := range slice {
		if e == elem {
			return true
		}
	}
	return false
}
