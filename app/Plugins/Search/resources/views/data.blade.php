@extends("App::app")

@section('title',__("app.search result",['search'=>"「".$q."]"]))
@section('description',__("app.search result",['search'=>"「".$q."]"]))

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-md-9">
                    <div class="row row-cards justify-content-center">
                        @if($page->count())
                            @foreach($page as $data)
                                <div class="col-md-12">
                                    <div class="border-0 card card-body">
                                        <div class="row">
                                            <div class="col-md-12">
                                                <div class="row">
                                                    <div class="col-auto">
                                                        <a href="/users/{{$data->user->username}}.html" class="avatar"
                                                           style="background-image: url({{super_avatar($data->user)}})"></a>
                                                    </div>
                                                    <div class="col">
                                                        <a href="/users/{{$data->user->username}}.html"
                                                           style="margin-bottom:0;text-decoration:none;"
                                                           class="card-title text-reset">{{$data->user->username}}</a>
                                                        <div style="margin-top:1px">{{__("app.Published on")}}
                                                            :{{$data->created_at}}</div>
                                                    </div>
                                                    <div class="col-auto">
                                                        @if($data->essence>0)
                                                            <div class="ribbon bg-green text-h3">
                                                                {{__("app.essence")}}
                                                            </div>
                                                        @endif
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="col-md-12">
                                                @if($data->comment_id)
                                                    <a href="{{$data->comments->topic->id}}.html" class="text-reset">
                                                        @endif
                                                        @if($data->topic_id)
                                                            <a href="{{$data->topic->id}}.html" class="text-reset">
                                                                @endif
                                                                <div class="row">
                                                                    <div class="col-md-12 markdown home-article">
                                                                        <span class="home-summary">{!! $data->content !!}</span>
                                                                    </div>
                                                                </div>
                                                            </a>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            @endforeach
                        @else
                            <div class="col-md-12">
                                <div class="border-0 card card-body">
                                    <div class="text-center card-title">{{__("app.No more results")}}</div>
                                </div>
                            </div>
                        @endif
                        {!! make_page($page) !!}
                    </div>
                </div>
                <div class="col-md-3">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
                            @include('App::index.right')
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
@section('scripts')
    <script src="/tabler/libs/apexcharts/dist/apexcharts.min.js"></script>
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
@endsection
