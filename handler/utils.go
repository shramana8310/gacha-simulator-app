package handler

import (
	"gacha-simulator/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

func getTranslationIndex(preferred []language.Tag, translationHolder model.TranslationHolder) int {
	var tags []language.Tag
	languageHolders := translationHolder.GetLanguageHolders()
	for i := 0; i < len(languageHolders); i++ {
		tag := language.Make(languageHolders[i].GetLanguage())
		tags = append(tags, tag)
	}
	matcher := language.NewMatcher(tags)
	_, i, _ := matcher.Match(preferred...)
	return i
}

func getPreferredLanguage(c *gin.Context) []language.Tag {
	preferred, _, err := language.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
	if err != nil {
		return []language.Tag{language.English}
	}
	return preferred
}
