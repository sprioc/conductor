package sql

import (
	"log"

	"github.com/lib/pq"
	"github.com/sprioc/composer/pkg/model"
)

// GetUser returns the fields of a user row into a User struct, including image references.
func GetUser(u int64) (model.User, error) {
	user := model.User{}
	err := db.Get(&user, "SELECT * FROM content.users WHERE id = $1", u)
	if err != nil {
		log.Println(err)
		return model.User{}, err
	}

	images, err := GetUserImages(u)
	if err != nil {
		log.Println(err)
		return model.User{}, err
	}
	user.Images = images

	favorites, err := userFavorites(u)
	if err != nil {
		log.Println(err)
		return model.User{}, err
	}
	user.Favorites = favorites

	return user, nil
}

func GetUsers(userIds []int64) ([]model.User, error) {
	users := []model.User{}
	for _, userId := range userIds {
		usr, err := GetUser(userId)
		if err != nil {
			log.Println(err)
			return []model.User{}, err
		}
		users = append(users, usr)
	}
	return users, nil
}

func GetUserImages(userId int64) ([]model.Image, error) {
	images := []int64{}
	err := db.Select(&images, `
	SELECT id FROM content.images
	WHERE user_id = $1`, userId)
	if err != nil {
		log.Println(err)
		return []model.Image{}, err
	}

	return GetImages(images)
}

func userFavorites(userId int64) ([]model.Image, error) {
	favorites := []int64{}
	err := db.Select(&favorites, `
	SELECT image_id FROM content.user_favorites
	WHERE user_id = $1`, userId)
	if err != nil {
		log.Println(err)
		return []model.Image{}, err
	}

	return GetImages(favorites)
}

func GetImages(imageIDS []int64) ([]model.Image, error) {
	images := []model.Image{}
	for _, imageId := range imageIDS {
		img, err := GetImage(imageId)
		if err != nil {
			log.Println(err)
			return []model.Image{}, err
		}
		images = append(images, img)
	}
	return images, nil
}

// GetImage takes an image ID and returns a image row into a Image struct including metadata
// and user data.
func GetImage(i int64) (model.Image, error) {
	img := model.Image{}
	err := db.Get(&img, `
	SELECT * FROM content.images AS images
	WHERE images.id = $1`, i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}

	img.Metadata, err = imageMetadata(i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}

	img.Landmarks, err = imageLandmarks(i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}

	img.Labels, err = imageLabels(i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}
	img.Tags, err = imageTags(i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}
	img.Colors, err = imageColors(i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}

	img.Stats, err = imageStats(i)
	if err != nil {
		log.Println(err)
		return model.Image{}, err
	}

	img.Source = imageSources(img.Shortcode, "content")

	return img, nil
}

func imageLandmarks(imageId int64) ([]model.Landmark, error) {
	landmarks := []model.Landmark{}
	rows, err := db.Query(`
	SELECT landmark.desc, landmark.location, bridge.score FROM content.image_landmark_bridge AS bridge
	JOIN content.landmarks AS landmark ON bridge.landmark_id = landmark.id
	WHERE bridge.image_id = $1`, imageId)
	if err != nil {
		log.Println(err)
		return landmarks, err
	}
	defer rows.Close()
	for rows.Next() {
		landmark := model.Landmark{}
		var loc string
		err = rows.Scan(&landmark.Description, &loc, &landmark.Score)
		if err != nil {
			log.Println(err)
		}
		log.Println(loc)
		landmarks = append(landmarks, landmark)
	}
	return landmarks, nil
}

func imageSources(shortcode, location string) model.ImageSource {
	var prefix = "https://images.sprioc.xyz/" + location + "/"
	var resourceBaseURL = prefix + shortcode
	return model.ImageSource{
		Raw:    resourceBaseURL,
		Large:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy",
		Medium: resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=1080&fit=max",
		Small:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=400&fit=max",
		Thumb:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=200&fit=max",
	}
}

func imageLabels(imageId int64) ([]model.Label, error) {
	labels := []model.Label{}
	err := db.Select(&labels, `
	SELECT description, score FROM content.image_label_bridge
	JOIN content.labels ON content.image_label_bridge.label_id = content.labels.id
	WHERE content.image_label_bridge.image_id = $1`,
		imageId)
	if err != nil {
		log.Println(err)
		return labels, err
	}
	return labels, nil
}

func imageTags(imageId int64) ([]string, error) {
	tags := []string{}
	err := db.Select(&tags, `
	SELECT description FROM content.image_tags AS tags
	JOIN content.image_tag_bridge AS bridge ON tags.id = content.image_tag_bridge.tag_id
	WHERE bridge.image_id = $1`,
		imageId)
	if err != nil {
		log.Println(err)
		return tags, err
	}
	return tags, nil
}

func imageStats(imageId int64) (model.ImageStats, error) {
	stats := model.ImageStats{}
	err := db.Get(&stats, `
	SELECT downloads, views, favorites FROM content.images
	WHERE id = $1`, imageId)
	if err != nil {
		log.Println(err)
		return stats, err
	}
	return stats, nil
}

func imageColors(imageId int64) ([]model.Color, error) {
	colors := []model.Color{}

	rows, err := db.Query(`
	SELECT red,green,blue, hue,saturation,val, shade, color, pixel_fraction, score from content.colors AS colors
	JOIN content.image_color_bridge AS bridge ON colors.id = content.image_color_bridge.color_id
	WHERE bridge.image_id = $1
	`, imageId)
	if err != nil {
		log.Println(err)
		return colors, err
	}

	defer rows.Close()
	for rows.Next() {
		color := model.Color{}
		err = rows.Scan(&color.SRGB.R, &color.SRGB.G, &color.SRGB.B,
			&color.HSV.H, &color.HSV.S, &color.HSV.V, &color.Shade, &color.ColorName,
			&color.PixelFraction, &color.Score)
		if err != nil {
			log.Println(err)
			return colors, err
		}
		color.Hex = color.SRGB.Hex()
		colors = append(colors, color)
	}
	return colors, nil
}

func imageMetadata(imageId int64) (model.ImageMetadata, error) {
	meta := model.ImageMetadata{}
	err := db.Get(&meta, `
	SELECT * FROM content.image_metadata
	WHERE image_id = $1`, imageId)
	if err != nil {
		log.Println(err)
		return meta, err
	}
	return meta, nil
}

func GetImageID(shortcode string) (int64, error) {
	var iID int64

	err := db.Get(&iID, "SELECT id FROM content.images WHERE shortcode = $1",
		shortcode)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return iID, nil
}

func GetRecentImages(limit int) ([]model.Image, error) {
	imageIds := []int64{}
	err := db.Select(&imageIds,
		`
	SELECT images.id
	FROM content.images AS images
	ORDER BY publish_time DESC LIMIT $1
		`,
		limit)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			log.Printf("%+v", err)
		}
		return []model.Image{}, err
	}
	return GetImages(imageIds)
}

func GetUserFollowed(userID int64) ([]model.User, error) {
	userIds := []int64{}
	err := db.Select(&userIds,
		`
	SELECT users.id
	FROM content.user_follows AS follows 
		JOIN content.users AS users ON id = follows.followed_id
	WHERE follows.user_id = $1
		`,
		userID)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			log.Printf("%+v", err)
		}
		return []model.User{}, err
	}
	return GetUsers(userIds)
}

func GetUserFavorites(userID int64) ([]model.Image, error) {
	imgs := []int64{}
	err := db.Select(&imgs,
		`
	SELECT images.id
	FROM content.user_favorites AS favs
		JOIN content.images AS images ON favs.image_id = images.id
	WHERE favs.user_id = $1
	ORDER BY publish_time DESC
		`,
		userID)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			log.Printf("%+v", err)
		}
		return []model.Image{}, err
	}
	return GetImages(imgs)
}

func GetFeaturedImages(limit int) ([]model.Image, error) {
	imgs := []int64{}
	err := db.Select(&imgs,
		`
	SELECT images.id
	FROM content.images AS images
	WHERE featured = TRUE 
	ORDER BY publish_time DESC LIMIT $1
		`,
		limit)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			log.Printf("%+v", err)
		}
		return []model.Image{}, err
	}
	return GetImages(imgs)
}

func GetImageRef(i string) (model.Ref, error) {
	ref := model.Ref{Collection: model.Images}
	err := db.Get(&ref.Id, "SELECT id FROM content.images WHERE shortcode = $1", i)
	if err != nil {
		log.Println(err)
		return model.Ref{}, err
	}
	return ref, nil
}

func GetUserRef(u string) (model.Ref, error) {
	ref := model.Ref{Collection: model.Users}
	err := db.Get(&ref.Id, "SELECT id FROM content.users WHERE username = $1", u)
	if err != nil {
		log.Println(err)
		return model.Ref{}, err
	}
	return ref, nil
}
