@extends("plugins.Core.app")

@section('title',"草稿预览 - ".$data->title)

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    <div class="row row-cards justify-content-center">
                        <div class="col-md-12">
                            <div class="border-0 card">
                                <div class="card-body topic">
                                    <div class="row">
                                        <div class="col-md-12" id="title">
                                            <h1 data-bs-toggle="tooltip" data-bs-placement="left" title="帖子标题">

                                                {{ $data->title }}
                                            </h1>
                                        </div>
                                        <hr class="hr-text" style="margin-top: 5px;margin-bottom: 5px">
                                        <div class="col-md-12" id="author">
                                            <div class="row">
                                                <div class="col-auto">
                                                    <a class="avatar" href="/users/{{ $data->user->username }}.html"
                                                       style="background-image: url({{ super_avatar($data->user) }})"></a>
                                                </div>
                                                <div class="col">
                                                    <div class="topic-author-name">
                                                        <a href="/users/{{ $data->user->username }}.html"
                                                           class="text-reset">{{ $data->user->username }}</a>
                                                    </div>
                                                    <div>创建于:{{ format_date($data->created_at) }}</div>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="col-md-12 vditor-reset" id="topic-content">
                                            {!! ShortCodeR()->handle($data->content) !!}
                                        </div>

                                    </div>
                                </div>
                            </div>

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
    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
@endsection
