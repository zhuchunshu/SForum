{{--<b>{{get_options('wealth_money_name', '余额') . ' >>> ' . get_options('wealth_golds_name', '金币')}}</b>--}}
@php($proportion = get_options('wealth_how_many_money_to_credit',get_options('wealth_how_many_money_to_golds','1')*get_options('wealth_how_many_golds_to_credit',10)))
兑换比例: 1 {{get_options('wealth_money_name', '余额')}} >>>  {{$proportion}} {{get_options('wealth_credit_name', '积分')}}
<div class="mb-2">
    @php($user = \App\Plugins\User\src\Models\User::query()->with('Options')->find(auth()->id()))
    @if(count(explode('.',$user->Options->money))>1)
        @php( $dc = \Hyperf\Stringable\Str::after((string)$user->Options->money,'.'))
        @php($dc = (int)\Hyperf\Stringable\Str::length($dc))
    @else
        @php($dc = 0)
    @endif
    当前 {{get_options('wealth_money_name', '余额')}}: <b class="text-red">{{$user->Options->money}}</b> {{get_options('wealth_money_unit_name', '元')}}，
    最多能兑换 <b class="text-red">{{intval($user->Options->money*$proportion)}}</b> {{get_options('wealth_credit_name', '积分')}}
    @if(round($user->Options->money - (intval($user->Options->money*$proportion))/$proportion,$dc)!=0)
        有 {{round($user->Options->money - (intval($user->Options->money*$proportion))/$proportion,$dc)}} {{get_options('wealth_money_unit_name', '元')}} 不能兑换
    @endif
</div>


<div class="mb-2">
    <label for="" class="form-label"></label>
    <input type="number" min="0" step="1" max="{{intval($user->Options->money*$proportion)}}" class="form-control" v-model="data.moneyTo_credit_num" placeholder="输入兑换的{{get_options('wealth_credit_name', '积分')}}数量" required>
</div>