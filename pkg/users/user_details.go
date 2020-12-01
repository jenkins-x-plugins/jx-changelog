package users

import (
	v1 "github.com/jenkins-x/jx-api/v3/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

type UserDetailService struct {
	cache map[string]*v1.UserDetails
}

func (s *UserDetailService) GetUser(login string) *v1.UserDetails {
	if s.cache == nil {
		s.cache = map[string]*v1.UserDetails{}
	}
	return s.cache[login]
}

func (s *UserDetailService) CreateOrUpdateUser(u *v1.UserDetails) error {
	if u == nil || u.Login == "" {
		return nil
	}

	log.Logger().Infof("CreateOrUpdateUser: %s <%s>", u.Login, u.Email)

	id := naming.ToValidName(u.Login)

	// check for an existing user by email
	existing := s.GetUser(id)
	if existing == nil {
		s.cache[id] = u
		return nil
	}
	if u.Email != "" {
		existing.Email = u.Email
	}
	if u.AvatarURL != "" {
		existing.AvatarURL = u.AvatarURL
	}
	if u.URL != "" {
		existing.URL = u.URL
	}
	if u.Name != "" {
		existing.Name = u.Name
	}
	if u.Login != "" {
		existing.Login = u.Login
	}
	return nil
}
