package coverletter

import (
	"job-bot/internal/cases"
	"job-bot/internal/entities"

	"context"
	"fmt"
)

type Applicant struct {
	Name  string
	Phone string
	Email string
}

type TemplateGenerator struct{ who Applicant }

func NewTemplateGenerator(a Applicant) *TemplateGenerator {
	return &TemplateGenerator{who: a}
}

func (g *TemplateGenerator) Generate(_ context.Context, v entities.Vacancy) (string, error) {
	return fmt.Sprintf(`Здравствуйте!

Меня заинтересовала ваша вакансия «%s» в компании %s. Уверен, что мой опыт и навыки будут полезны вашей команде.

Буду рад обсудить детали на собеседовании.

С уважением,
%s
%s | %s`, v.Title, v.Company, g.who.Name, g.who.Phone, g.who.Email), nil
}

var _ cases.LetterGenerator = (*TemplateGenerator)(nil)
