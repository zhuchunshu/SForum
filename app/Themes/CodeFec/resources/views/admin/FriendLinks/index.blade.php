@extends('app')

@section('title','友链管理')

@section('content')
<div class="row row-cards">
    @if(!$page->count())
        <div class="col-md-12">
            <div class="card card-body empty">
                <div class="empty-header">403</div>
                <p class="empty-title">暂无数据</p>
                <p class="empty-subtitle text-muted">
                    没有数据啦，请点击右上角创建
                </p>
            </div>
        </div>
    @else
        <div class="col-lg-12">
            <div class="row row-cards">
                @foreach($page as $data)
                    <div class="col-6 col-md-4 col-lg-3 ">
                        <a href="/admin/setting/friend_links/{{$data->id}}/edit" class="card card-link">
                            <div class="card-body">
                                <h3 class="card-title">{{$data->name}}</h3>
                                <span class="text-muted">{{\Hyperf\Stringable\Str::limit($data->link,80)}}</span>
                            </div>
                        </a>
                    </div>
                @endforeach
            </div>
        </div>
        <div class="col-12">
            {!! make_page($page) !!}
        </div>
    @endif
</div>
@endsection

@section('headerBtn')
    <a class="btn btn-primary" href="/admin/setting/friend_links/create">新增友链</a>
@endsection