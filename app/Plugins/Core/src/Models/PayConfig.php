<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
/**
 * @property string $name 
 * @property string $value 
 */
class PayConfig extends Model
{
    public bool $timestamps = false;
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'pay_config';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['name', 'value'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = [];
}