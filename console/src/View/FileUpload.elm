module View.FileUpload exposing (Model, Msg, bytes, init, mime, update, view)

import Bytes exposing (Bytes)
import Bytes.Encode as BE
import Element exposing (Element)
import File exposing (File)
import File.Select
import Html
import Html.Attributes as Attr
import Proto.Api as PB
import Task
import View.Atom.Button


type Msg
    = LoadRequest
    | Selected File
    | ToBytes Bytes
    | ToUrl String


type alias Model =
    { name : String
    , mime : String
    , preview : Maybe String
    , bytes : Bytes
    }


init : Model
init =
    { name = ""
    , mime = ""
    , preview = Nothing
    , bytes = BE.encode <| BE.string ""
    }


mime : Model -> String
mime model =
    model.mime


bytes : Model -> Bytes
bytes model =
    model.bytes


mimes : List String
mimes =
    [ "image/jpeg", "image/png", "image/gif" ]


update : (Msg -> msg) -> Msg -> Model -> ( Model, Cmd msg )
update toMsg msg model =
    case msg of
        LoadRequest ->
            ( model
            , File.Select.file mimes (toMsg << Selected)
            )

        Selected file ->
            let
                mime_ =
                    File.mime file
            in
            ( { model
                | name = File.name file
                , mime = mime_
              }
            , Cmd.batch
                [ if List.member mime_ mimes then
                    Task.perform (toMsg << ToUrl) <| File.toUrl file

                  else
                    Cmd.none
                , Task.perform (toMsg << ToBytes) <| File.toBytes file
                ]
            )

        ToUrl url ->
            ( { model | preview = Just url }
            , Cmd.none
            )

        ToBytes bs ->
            ( { model | bytes = bs }
            , Cmd.none
            )


maybe : b -> (a -> b) -> Maybe a -> b
maybe b f =
    Maybe.withDefault b << Maybe.map f


maybe_ : Maybe a -> b -> (a -> b) -> b
maybe_ ma b f =
    maybe b f ma


view :
    (Msg -> msg)
    -> msg
    -> Model
    -> Element msg
view toMsg submitMsg model =
    Element.column []
        [ Element.row []
            [ View.Atom.Button.button (toMsg LoadRequest) "File Select"
            , Element.text model.name
            ]
        , maybe_ model.preview Element.none <|
            \prev ->
                Element.column []
                    [ View.Atom.Button.updateButton submitMsg "Upload"
                    , Element.html <| Html.img [ Attr.src prev, Attr.style "width" "400px" ] []
                    ]
        ]
