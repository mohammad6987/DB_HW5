package models

import (
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Paper struct {
    ID               primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
    Title            string               `bson:"title" json:"title"`
    Authors          []string             `bson:"authors" json:"authors"`
    Abstract         string               `bson:"abstract" json:"abstract"`
    PublicationDate  primitive.DateTime   `bson:"publication_date" json:"publication_date"`
    JournalConference string              `bson:"journal_conference" json:"journal_conference"`
    Keywords         []string             `bson:"keywords" json:"keywords"`
    UploadedBy       primitive.ObjectID   `bson:"uploaded_by" json:"uploaded_by"`
    Views            int                  `bson:"views" json:"views"`
}

type Citation struct {
    ID           primitive.ObjectID `bson:"_id,omitempty"`
    PaperID      primitive.ObjectID `bson:"paper_id"`
    CitedPaperID primitive.ObjectID `bson:"cited_paper_id"`
}
