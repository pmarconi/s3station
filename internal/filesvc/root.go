package filesvc

import "station/internal/storage"

func (s *Service) abs(key string) string {
	return storage.AbsUserKey(s.root, key)
}

func (s *Service) rel(key string) string {
	return storage.RelUserKey(s.root, key)
}
