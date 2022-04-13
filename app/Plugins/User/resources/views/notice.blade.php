@extends("App::app")

@section('title', '我的通知')

@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Overview
                        </div>
                        <h2 class="page-title">
                            通知列表
                        </h2>
                    </div>
                    <div class="col-auto">
					<button user-click="notice_allread" class="btn btn-danger">一键清空未读通知</button>
					</div>

                </div>
            </div>
        </div>
    </div>
@endsection

@section('content')
    <div class="col-md-12">
        <div class="row row-cards justify-content-center">
            @if($page->count())
                @foreach($page as $value)
                    <div class="col-md-12">
                        <div class="border-0 card">
                            <div class="card-header">
                                <div class="card-title">{{$value->title}}</div>
                            </div>
                            <div class="card-body">
                                {!! $value->content !!}
                            </div>
                            <div class="card-footer">
                                <div class="row">
                                    <div class="col">
                                        <span data-bs-toggle="tooltip" data-bs-placement="top" title="{{$value->created_at}}" class="badge badge-outline text-blue">{{format_date($value->created_at)}}</span>
                                    </div>
                                    <div class="col-auto">
                                        @if($value->action)
                                            <a user-click="notice_action" notice-href="{{$value->action}}" notice-id="{{$value->id}}" class="btn btn-primary">查看</a>
                                        @endif
                                        <button user-click="notice_read" notice-id="{{$value->id}}" class="btn btn-danger">已读</button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                @endforeach
            @else
                <div class="col-md-12">
                    <div class="border-0 card">
                        <div class="card-body">
                            <h3 class="card-title">暂无通知</h3>
                        </div>
                    </div>
                </div>
            @endif
        </div>
        <div class="mt-3">
            {!! make_page($page) !!}
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection
