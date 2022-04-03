@extends('Core::app')

@section('title','【'.request()->input('q').'】的搜索结果')
@section('description','为你找到'.$data['nbHits'].'条【'.request()->input('q').'】的搜索结果')

@section('header')
    <div class="container-xl mt-3">
        <!-- Page title -->
        <div class="page-header d-print-none">
            <div class="row align-items-center">
                <div class="col">
                    <h2 class="page-title">
                        【{{request()->input('q')}}】的搜索结果
                    </h2>
                    查询耗时:{{$data['processingTimeMs']}}毫秒
                </div>
            </div>
        </div>
    </div>
@endsection

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">

                    <div class="row row-cards">

                        @if(request()->input('page',1)<=$page_num)
                            @foreach($data['hits'] as $value)
                                <div class="col-md-12">
                                    <div class="card card-link" href="{{url($value['url'])}}">
                                        <div class="card-body search-highlight">
                                            <div class="row">
                                                <div class="col-auto">
                                                    <a href="{{url('/users/'.$value['username'].".html")}}">
                                                        <span class="align-middle avatar"
                                                              style="background-image: url({{$value['avatar']}})"></span>
                                                    </a>
                                                </div>

                                                <div class="col">
                                                    <h3 class="card-title"><a
                                                                href="{{url($value['url'])}}">{!!$value['_formatted']['title']!!}</a>
                                                    </h3>
                                                    类型:{{$value['type']}} | <a
                                                            href="{{url($value['url'])}}">{{url($value['url'])}}</a>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            @endforeach

                        @else
                            <div class="col-md-12">
                                <h3 class="card-title">无【{{request()->input('q')}}】的搜索结果</h3>
                            </div>

                        @endif


                        @if($page_num>1)
                            <div class="col-md-12">
                                <div class="col-12">
                                    <div class="card">
                                        <div class="card-body">
                                            <ul class="pagination ">

                                                @if(request()->input('page',1)>1)
                                                    <li class="page-item page-prev">
                                                        <a class="page-link"
                                                           href="?{{core_http_build_query(request()->all(),['page' => request()->input('page',1)-1])}}"
                                                           tabindex="-1">
                                                            <div class="page-item-subtitle">previous</div>
                                                            <div class="page-item-title">上一页</div>
                                                        </a>
                                                    </li>

                                                @else
                                                    <li class="page-item page-prev disabled">
                                                        <a class="page-link" href="#" tabindex="-1"
                                                           aria-disabled="true">
                                                            <div class="page-item-subtitle">previous</div>
                                                            <div class="page-item-title">上一页</div>
                                                        </a>
                                                    </li>
                                                @endif
                                                @if(request()->input('page',1)<$page_num)

                                                    <li class="page-item page-next">
                                                        <a class="page-link"
                                                           href="?{{core_http_build_query(request()->all(),['page' => request()->input('page',1)+1])}}">
                                                            <div class="page-item-subtitle">next</div>
                                                            <div class="page-item-title">下一页</div>
                                                        </a>
                                                    </li>
                                                @else
                                                    <li class="page-item page-next disabled">
                                                        <a class="page-link" href="#" tabindex="-1"
                                                           aria-disabled="true">
                                                            <div class="page-item-subtitle">next</div>
                                                            <div class="page-item-title">下一页</div>
                                                        </a>
                                                    </li>
                                                @endif
                                            </ul>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        @endif

                    </div>

                </div>

                <div class="col-md-5">
                    <div class="row row-cards">
                        <div class="col-md-10">
                            <div class="row row-cards">
                                <div class="col-md-12">
                                    <div class="card">
                                        <div class="card-status-top bg-primary"></div>
                                        <div class="card-body">
                                            <h3 class="card-title">搜索</h3>
                                            <form action="/search">
                                                <div class="mb-3">
                                                    <input type="text" class="form-control" name="q"
                                                           value="{{request()->input('q')}}" placeholder="Search...">
                                                </div>
                                                <button class="btn btn-primary">搜索</button>
                                            </form>
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

@section('css')
<style>
.search-highlight em{
    color:red;
    text-shadow: h-shadow v-shadow blur-radius color|none;
    font-weight: bold
}
</style>
@endsection