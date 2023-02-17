{{--<b>{{get_options('wealth_money_name', '余额') . ' >>> ' . get_options('wealth_golds_name', '金币')}}</b>--}}
兑换比例: 1 {{get_options('wealth_money_name', '余额')}} >>>  {{get_options('wealth_how_many_money_to_golds','1')}} {{get_options('wealth_golds_name', '金币')}}
<div class="mb-2">
    @php($user = \App\Plugins\User\src\Models\User::query()->with('Options')->find(auth()->id()))
    @if(count(explode('.',$user->Options->money))>1)
        @php( $dc = \Hyperf\Utils\Str::after((string)$user->Options->money,'.'))
        @php($dc = (int)\Hyperf\Utils\Str::length($dc))
    @else
        @php($dc = 0)
    @endif
    当前 {{get_options('wealth_money_name', '余额')}}: <b class="text-red">{{$user->Options->money}}</b> {{get_options('wealth_money_unit_name', '元')}}，
    最多能兑换 <b class="text-red">{{intval($user->Options->money*get_options('wealth_how_many_money_to_golds','1'))}}</b> {{get_options('wealth_golds_name', '金币')}}
    @if(round((float)(int)$user->Options->money -(float)(int)(intval($user->Options->money*get_options('wealth_how_many_money_to_golds','1')))/get_options('wealth_how_many_money_to_golds','1'),$dc)!=0)
        有 {{round((float)(int)$user->Options->money - (float)(int)(intval($user->Options->money*get_options('wealth_how_many_money_to_golds','1')))/get_options('wealth_how_many_money_to_golds','1'),$dc)}} {{get_options('wealth_money_unit_name', '元')}} 不能兑换
    @endif
</div>


<div class="mb-2">
    <label for="" class="form-label"></label>
    <input type="number" min="0" step="1" max="{{(float)(int)intval($user->Options->money*get_options('wealth_how_many_money_to_golds','1'))}}" class="form-control" v-model="data.moneyTo_golds_num" placeholder="输入兑换的{{get_options('wealth_golds_name', '金币')}}数量" required>
</div>