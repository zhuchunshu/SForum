@extends("App::app")
@section('title','资产变更记录')
@section('content')

    <div class="row row-cards">
        <div class="col-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">资产变更记录</h3>
                    <div class="card-actions">
                        <a href="/users/{{auth()->id()}}.html?m=users_home_menu_12">返回任务列表</a>
                    </div>
                </div>
                <div class="card-body table-responsive">
                    <table class="table table-vcenter">
                        <thead>
                        <tr>
                            <th>#</th>
                            <th class="text-nowrap">变更前</th>
                            <th class="text-nowrap">变更后</th>
                            <th class="text-nowrap">资产类型</th>
                            <th class="text-nowrap">变化</th>
                            <th class="text-nowrap">备注</th>
                            <th class="text-nowrap">创建时间</th>
                        </tr>
                        </thead>
                        <tbody>
                        @if(!$page->count())
                            <tr>
                                <th class="text-center" colspan="6">暂无结果</th>
                            </tr>
                        @else
                            @foreach($page as $data)
                                <tr>
                                    <th>{{$data->id}}</th>
                                    <td>{{$data->original}}</td>
                                    <td>{{$data->cash}}</td>
                                    <td>{{get_options('wealth_'.$data->type.'_name')?:$data->type}}</td>
                                    <td>{{(!is_negative($data->change)===false)?"":"+" }}{{$data->change}}</td>
                                    <td>@if($data->remark) {{$data->remark}} @else 无备注 @endif</td>
                                    <td>{{$data->created_at}}</td>
                                </tr>
                            @endforeach
                        @endif
                        </tbody>
                    </table>
                    <div class="mt-2">
                        @if($page->count())
                            {!! make_page($page) !!}
                        @endif
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection

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
                            资产变更记录
                        </h2>
                    </div>

                    <div class="col-auto">
                        <a href="/users/{{auth()->id()}}.html" class="btn btn-primary">个人中心</a>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection