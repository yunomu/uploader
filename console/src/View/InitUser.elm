module View.InitUser exposing (Model, Msg, init, update, view)

import Element exposing (Element)
import Element.Input
import Proto.Api
import View.Atom.Button
import View.Atom.DescriptiveList


type alias Model =
    { name : String
    }


type Msg
    = ChangeName String


init : Model
init =
    { name = ""
    }


mk : Model -> Proto.Api.InitUserRequest
mk model =
    { name = model.name
    }


template : String -> Element msg -> Element msg -> Element msg
template caption form operation =
    Element.column
        [ Element.paddingXY 20 0
        , Element.spacing 30
        ]
        [ Element.text caption
        , form
        , operation
        ]


view :
    { commit : Proto.Api.InitUserRequest -> msg
    , cancel : msg
    , toMsg : Msg -> msg
    }
    -> Model
    -> Element msg
view msgs model =
    template "Init User"
        (View.Atom.DescriptiveList.view
            [ { header = "User name"
              , body =
                    Element.Input.text []
                        { onChange = msgs.toMsg << ChangeName
                        , text = model.name
                        , placeholder = Nothing
                        , label = Element.Input.labelHidden "name"
                        }
              }
            ]
        )
        (Element.row [ Element.spacing 20 ]
            [ View.Atom.Button.updateButton (msgs.commit <| mk model) "Submit"
            , View.Atom.Button.button msgs.cancel "Cancel"
            ]
        )


update : Msg -> Model -> Model
update msg model =
    case msg of
        ChangeName name ->
            { model | name = name }
