module View.Files exposing (Model, continuationToken, init, view)

import Bytes exposing (Bytes)
import Element exposing (Element)
import File exposing (File)
import File.Select
import Html
import Html.Attributes as Attr
import Proto.Api as PB
import Task
import View.Atom.Button
import View.Atom.Table


type alias Model =
    { files : List PB.Object
    , continuationToken : Maybe String
    }


init : List PB.Object -> Maybe String -> Model
init files ctoken =
    { files = files
    , continuationToken = ctoken
    }


continuationToken : Model -> Maybe String
continuationToken model =
    model.continuationToken


mimes : List String
mimes =
    [ "image/jpeg", "image/png", "image/gif" ]


maybe : b -> (a -> b) -> Maybe a -> b
maybe b f =
    Maybe.withDefault b << Maybe.map f


maybe_ : Maybe a -> b -> (a -> b) -> b
maybe_ ma b f =
    maybe b f ma


view :
    { upload : msg
    , selected : String -> msg
    , delete : String -> Int -> msg
    }
    -> Model
    -> Element msg
view msgs model =
    Element.column []
        [ Element.text "Files"
        , View.Atom.Button.button msgs.upload "Upload"
        , View.Atom.Table.view model.files
            (\_ _ -> False)
            [ { header = "Key"
              , view = \i r -> Element.text r.key
              }
            , { header = "Content-Type"
              , view = \i r -> Element.text r.contentType
              }
            , { header = ""
              , view =
                    \i r ->
                        Element.row []
                            [ View.Atom.Button.button (msgs.selected r.key) "Show"
                            , View.Atom.Button.updateButton (msgs.delete r.key r.timestamp) "Delete"
                            ]
              }
            ]
        , View.Atom.Button.button msgs.upload "Upload"
        ]
