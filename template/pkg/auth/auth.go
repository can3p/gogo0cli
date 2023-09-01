package auth

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"<projectrepo>/pkg/admin"
	"<projectrepo>/pkg/model/core"
	"<projectrepo>/pkg/pgsession"
	"<projectrepo>/pkg/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	userkey = "user"
)

func Auth(c *gin.Context, db *sqlx.DB) {
	session := sessions.Default(c)
	user := session.Get(userkey)

	if user == nil {
		c.Next()

		return
	}

	if err := pgsession.SetUser(c, db, user.(string)); err != nil {
		log.Printf("Failed to save user to pgsession, auth won't work as expected: %s", err)
	}

	c.Next()
}

func EnforceAuth(c *gin.Context) {
	userData := GetUserData(c)

	if !userData.IsLoggedIn {
		c.Redirect(http.StatusFound, "/")
		c.Abort()
		return
	}

	c.Next()
}

func Login(c *gin.Context, db boil.ContextExecutor, email string, password string) error {
	session := sessions.Default(c)
	h := pgsession.HashUserPwd(email, password)

	user, err := core.Users(
		core.UserWhere.Email.EQ(email),
		core.UserWhere.Pwdhash.EQ(null.StringFrom(h)),
		core.UserWhere.EmailConfirmedAt.IsNotNull(),
	).One(context.TODO(), db)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.Errorf("Bad credentials")
		}

		panic(err)
	}

	session.Set(userkey, user.ID)

	if err := session.Save(); err != nil {
		return errors.Wrapf(err, "Failed to save session")
	}

	return nil
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	session.Delete(userkey)
	if err := session.Save(); err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.Redirect(http.StatusFound, "/")
	c.Abort()
}

type UserData struct {
	User       *pgsession.User
	DBUser     *core.User
	IsLoggedIn bool
}

func GetUserData(c *gin.Context) UserData {
	var out UserData

	u := pgsession.GetUser(c)

	out.IsLoggedIn = u != nil
	out.User = u
	if u != nil {
		out.DBUser = u.DBUser
	}

	return out
}

func AddFlash(c *gin.Context, flash interface{}, vars ...string) {
	session := sessions.Default(c)

	session.AddFlash(flash, vars...)
	if err := session.Save(); err != nil {
		log.Printf("Failed to save session: %v", err)
	}
}

func GetFlashes(c *gin.Context, vars ...string) []interface{} {
	session := sessions.Default(c)

	flashes := session.Flashes(vars...)

	if len(flashes) != 0 {
		if err := session.Save(); err != nil {
			log.Printf("error in flashes saving session: %v", err)
		}
	}

	return flashes
}

func Signup(ctx context.Context, db *sqlx.DB, email string, password string, attribution string) (*core.User, error) {
	if password == "" || email == "" {
		return nil, errors.Errorf("Not enough data")
	}

	u := &core.User{
		ID:                uuid.NewString(),
		Email:             email,
		Pwdhash:           null.StringFrom(pgsession.HashUserPwd(email, password)),
		EmailConfirmSeed:  null.StringFrom(uuid.NewString()),
		SignupAttribution: null.NewString(attribution, attribution != ""),
	}

	err := util.Transact(db, func(tx *sql.Tx) error {

		if err := u.Insert(ctx, tx, boil.Infer()); err != nil {
			return err
		}

		go admin.NotifyNewUser(u)

		return nil
	})

	return u, err
}
