package db

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/pbkdf2"
)

var db *sql.DB

// Start is the database package launch method
// it enters or fetches the data required for the database
func Start() {
	/*
	 * allow user to enter db data
	 * used instead of environment variables
	 * if there are none
	 * since the service is open source
	 */
	var (
		uname string
		pw    string
		name  string
	)
	if os.Getenv("DB_UNAME") == "" && os.Getenv("DB_NAME") == "" {
		reader := bufio.NewReader(os.Stdin)
		color.Cyan("Enter db user name: ")
		uname, _ = reader.ReadString('\n')

		color.Cyan("Enter db pw: ")
		pw, _ = reader.ReadString('\n')

		color.Cyan("Enter db name: ")
		name, _ = reader.ReadString('\n')
	} else {
		uname = os.Getenv("DB_UNAME")
		pw = os.Getenv("DB_PW")
		name = os.Getenv("DB_NAME")
	}

	var err error
	db, err = sql.Open("postgres",
		"user="+uname+
			" password="+pw+
			" dbname="+name+
			" sslmode=disable")

	if err != nil {
		color.Red("ERR: pdb.go Init() => PostgreSQL config could not be established.")
		color.Red(err.Error())
	}

	/*
	  // test connection
	  err = db.Ping()
	  if err != nil{ // connection not successful
	    color.Red("ERR: pdb.go Init() => Database connection not working.")
	    color.Red(err.Error())
	  }else{ // connection successful
	    /*
	     * variables for storing the user data
	     *
	    var (
	      id int
	      uname string
	      pw string
	    )

	    /*
	     * run query
	     *
	    rows, err := db.Query(`select user_id, user_name, user_pw from "ITUser"`)
	    if err != nil{
	      color.Red("ERR: pdb.go Init() => Query could not be executed.")
	      color.Red(err.Error())
	    }

	    /*
	     * close database connection at the
	     * end of the enclosing function
	     *
	    defer rows.Close()

	    /*
	     * .Next() prepares the next data column for reading
	     * .Scan(values) transfers the data to the given variables
	     *
	    for rows.Next() {
	      err := rows.Scan(&id, &uname, &pw)
	      if err != nil {
	        color.Red("ERR: pdb.go Init() => Fetched values could not be scanned.")
	        color.Red(err.Error())
	      }
	      log.Println(id, uname, pw)
	    }

	    err = rows.Err()
	    if err != nil {
	      color.Red("ERR: pdb.go Init() => An error occured.")
	      color.Red(err.Error())
	    }
	  }
	*/
}

func CheckUserCredentials(ue string, pwd string) (bool, error) {
	rows, err := db.Query("select user_name, user_email, user_pw, user_hash from imgturtle.user where user_name='" + ue + "' or user_email='" + ue + "'")
	if err != nil {
		color.Red("ERR@pdb.go@CheckUserCredentials() => %s", err.Error())
		return false, err
	}

	if rows != nil {
		defer rows.Close()

		var (
			fUname string
			fEmail string
			fPw    string
			fHash  string
		)
		if rows.Next() {
			err := rows.Scan(&fUname, &fEmail, &fPw, &fHash)
			if err != nil {
				color.Red("ERR: pdb.go CheckUserCredentials => Fetched values could not be scanned.")
				color.Red(err.Error())
				return false, err
			}
			if fPw == string(pbkdf2.Key([]byte(pwd), []byte(fHash), 4096, 32, sha1.New)) {
				color.Green("User %s entered a valid password.", fUname)
				return true, nil
			}
			color.Red("User %s entered an invalid password.", fUname)
			return false, errors.New("Incorrect password.")
		} else {
			color.Green("User %s could not be found.", ue)
			return false, errors.New("No such user.")
		}
	}
	return false, nil
}

// InsertNewUser handles the database part of the process of
// registering a new user
func InsertNewUser(uname string, pwd string, email string) error {
	rows, err := db.Query("select user_name, user_email from imgturtle.user where user_name='" + uname + "' or user_email='" + email + "'")
	if err != nil {
		color.Red("ERR@pdb.go@InsertNewUser() => %s", err.Error())
	}

	if rows != nil {
		defer rows.Close()

		var (
			funame string
			femail string
		)
		for rows.Next() {
			err := rows.Scan(&funame, &femail)
			if err != nil {
				color.Red("ERR: pdb.go InsertNewUser() => Fetched values could not be scanned.")
				color.Red(err.Error())
				return err
			}
			if funame == uname && femail == email {
				return errors.New("User name '" + uname + "' and e-mail address '" + email + "' in use.")
			} else if funame == uname {
				return errors.New("User name '" + uname + "' in use.")
			} else if femail == email {
				return errors.New("E-mail address '" + email + "'in use.")
			}
		}
	}

	b := make([]byte, 32)
	rand.Read(b)
	salt := fmt.Sprintf("%x", b)

	epw := pbkdf2.Key([]byte(pwd), []byte(salt), 4096, 32, sha1.New)

	stmt, err := db.Prepare("INSERT INTO imgturtle.user(user_name,user_pw,user_email,user_hash) VALUES($1,$2,$3,$4)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(uname, epw, email, salt)
	if err != nil {
		return err
	}

	return nil
}