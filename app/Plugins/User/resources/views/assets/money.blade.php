@extends("App::app")
@section('title','我的余额')
@section('content')

    <div class="row row-cards">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">账单明细</h3>
                <div class="card-actions">
                    <a href="/users/{{auth()->data()->username}}.html?m=users_home_menu_8">充值</a>
                </div>
            </div>
            <div class="card-body">
                <table class="table table-responsive">
                    <thead>
                    <tr>
                        <th>#</th>
                        <th class="text-nowrap">变更前余额</th>
                        <th class="text-nowrap">变更后余额</th>
                        <th class="text-nowrap">绑定订单</th>
                        <th class="text-nowrap">备注</th>
                        <th class="text-nowrap">创建时间</th>
                    </tr>
                    </thead>
                    <tbody>
                    @if(!$page->count())
                        <tr>
                            <th>暂无结果</th>
                            <td>暂无结果</td>
                            <td>暂无结果</td>
                            <td>暂无结果</td>
                            <td>暂无结果</td>
                            <td>暂无结果</td>
                        </tr>
                    @else
                        @foreach($page as $data)
                            <tr>
                                <th>{{$data->id}}</th>
                                <td>{{$data->original}} {{get_options('wealth_money_unit_name','元')}}</td>
                                <td>{{$data->cash}} {{get_options('wealth_money_unit_name','元')}}</td>
                                <td>@if($data->order_id) <a href="/user/order/{{$data->order_id}}.order">{{$data->order_id}}</a> @else 未绑定订单 @endif</td>
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
                            {{get_options('wealth_money_name','余额')}}明细
                        </h2>
                    </div>

                    <div class="col-auto">
                        <a href="/users/{{auth()->data()->username}}.html" class="btn btn-primary">个人中心</a>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection