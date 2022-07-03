package repo

func (r * Repository) Emport()*SerializableRepository {
	sr := &SerializableRepository{
		Owner: r.Owner,
		Name: r.Name,
		Commits: r.hashToManifestPaths,
	}
	return sr
}

