package validate

func isValidDAG(dag map[string][]string) []string {
	in_degree := map[string]int{}
	for node := range dag {
		in_degree[node] = 0
	}
	for _, children := range dag {
		for _, child := range children {
			in_degree[child]++
		}
	}
	queue := []string{}
	for node, degree := range in_degree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	visited := map[string]bool{}
	for node := range in_degree {
		visited[node] = false
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if visited[node] {
			continue
		}
		visited[node] = true
		for _, child := range dag[node] {
			in_degree[child]--
			if in_degree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	not_visited := []string{}
	for node, v := range visited {
		if !v {
			not_visited = append(not_visited, node)
		}
	}
	return not_visited
}
