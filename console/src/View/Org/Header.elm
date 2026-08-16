module View.Org.Header exposing (Model, Msg(..), init, update, view)

import Element exposing (Element)
import Element.Border as Border


edges =
    { top = 0
    , bottom = 0
    , left = 0
    , right = 0
    }


type Msg
    = UpdateUser (Maybe String)


type alias Model =
    { loginFormURL : String
    , user : Maybe String
    }


init : String -> Maybe String -> Model
init loginFormURL user =
    { loginFormURL = loginFormURL
    , user = user
    }


view : Model -> Element msg
view model =
    Element.row
        [ Element.width Element.fill
        , Element.padding 5
        , Border.widthEach { edges | bottom = 1 }
        ]
        [ Element.link [ Element.alignLeft ]
            { url = "/"
            , label = Element.text "Blog Console"
            }
        , Element.el [ Element.alignRight ] <|
            case model.user of
                Just user ->
                    Element.text <| "User: " ++ user

                Nothing ->
                    Element.link []
                        { url = model.loginFormURL
                        , label = Element.text "Login/Signup"
                        }
        ]


update : Msg -> Model -> Model
update msg model =
    case msg of
        UpdateUser user ->
            { model | user = user }
