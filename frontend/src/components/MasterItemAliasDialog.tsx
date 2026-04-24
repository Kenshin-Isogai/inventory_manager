import { type FormEvent, useMemo, useState } from 'react'
import { useSWRConfig } from 'swr'
import { Loader2, PackagePlus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ItemCombobox } from '@/components/ui/item-combobox'
import { createMasterItem, upsertMasterAlias } from '@/lib/additionalApi'
import type { MasterDataSummaryResponse, MasterItemRecord, SupplierAliasSummary } from '@/types'

type Mode = 'item-with-alias' | 'alias-only'

type MasterItemAliasDialogProps = {
  open: boolean
  mode?: Mode
  masterData?: MasterDataSummaryResponse
  masterItems?: MasterItemRecord[]
  initialItemNumber?: string
  initialDescription?: string
  initialManufacturer?: string
  initialSupplierId?: string
  initialSupplierAliasNumber?: string
  initialItemId?: string
  onOpenChange: (open: boolean) => void
  onCompleted?: (result: { item?: MasterItemRecord; alias?: SupplierAliasSummary }) => void
}

const NO_SUPPLIER = '__no_supplier__'

export function MasterItemAliasDialog({
  open,
  mode = 'item-with-alias',
  masterData,
  masterItems = [],
  initialItemNumber = '',
  initialDescription = '',
  initialManufacturer = '',
  initialSupplierId = '',
  initialSupplierAliasNumber = '',
  initialItemId = '',
  onOpenChange,
  onCompleted,
}: MasterItemAliasDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open && (
        <MasterItemAliasForm
          key={`${mode}-${initialItemNumber}-${initialItemId}-${initialSupplierAliasNumber}`}
          mode={mode}
          masterData={masterData}
          masterItems={masterItems}
          initialItemNumber={initialItemNumber}
          initialDescription={initialDescription}
          initialManufacturer={initialManufacturer}
          initialSupplierId={initialSupplierId}
          initialSupplierAliasNumber={initialSupplierAliasNumber}
          initialItemId={initialItemId}
          onCancel={() => onOpenChange(false)}
          onCompleted={(result) => {
            onCompleted?.(result)
            onOpenChange(false)
          }}
        />
      )}
    </Dialog>
  )
}

function MasterItemAliasForm({
  mode,
  masterData,
  masterItems,
  initialItemNumber,
  initialDescription,
  initialManufacturer,
  initialSupplierId,
  initialSupplierAliasNumber,
  initialItemId,
  onCancel,
  onCompleted,
}: Required<Pick<MasterItemAliasDialogProps, 'mode' | 'masterItems' | 'initialItemNumber' | 'initialDescription' | 'initialManufacturer' | 'initialSupplierId' | 'initialSupplierAliasNumber' | 'initialItemId'>> & {
  masterData?: MasterDataSummaryResponse
  onCancel: () => void
  onCompleted: (result: { item?: MasterItemRecord; alias?: SupplierAliasSummary }) => void
}) {
  const { mutate } = useSWRConfig()
  const [itemId, setItemId] = useState(initialItemId)
  const [itemNumber, setItemNumber] = useState(initialItemNumber)
  const [description, setDescription] = useState(initialDescription)
  const [manufacturerKey, setManufacturerKey] = useState(normalizeKey(initialManufacturer))
  const [categoryKey, setCategoryKey] = useState(masterData?.categories[0]?.key ?? 'misc')
  const [supplierId, setSupplierId] = useState(initialSupplierId)
  const [supplierAliasNumber, setSupplierAliasNumber] = useState(initialSupplierAliasNumber || initialItemNumber)
  const [unitsPerOrder, setUnitsPerOrder] = useState(1)
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const itemOptions = useMemo(
    () => masterItems.map((item) => ({
      itemId: item.id,
      itemNumber: item.itemNumber,
      description: item.description,
      manufacturer: item.manufacturerKey,
      category: item.categoryKey,
    })),
    [masterItems],
  )
  const selectedSupplierId = supplierId || NO_SUPPLIER
  const isNewItemMode = mode === 'item-with-alias'

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')

    if (isNewItemMode && (!itemNumber.trim() || !description.trim() || !manufacturerKey.trim() || !categoryKey.trim())) {
      setError('Item number, description, manufacturer, and category are required.')
      return
    }
    if (!isNewItemMode && !itemId) {
      setError('Select a canonical item for this alias.')
      return
    }
    if ((supplierId || supplierAliasNumber.trim()) && (!supplierId || !supplierAliasNumber.trim())) {
      setError('Supplier and supplier alias must both be set to create an alias.')
      return
    }

    setIsSubmitting(true)
    try {
      const item = isNewItemMode
        ? await createMasterItem({
            itemNumber: itemNumber.trim(),
            description: description.trim(),
            manufacturerKey: normalizeKey(manufacturerKey),
            categoryKey: normalizeKey(categoryKey),
            defaultSupplierId: supplierId,
            note: note.trim(),
            lifecycleStatus: 'active',
          })
        : undefined

      const resolvedItemId = item?.id ?? itemId
      const alias = supplierId && supplierAliasNumber.trim()
        ? await upsertMasterAlias({
            itemId: resolvedItemId,
            supplierId,
            supplierItemNumber: supplierAliasNumber.trim(),
            unitsPerOrder: Math.max(1, unitsPerOrder),
          })
        : undefined

      await Promise.all([mutate('master-data'), mutate('master-items'), mutate('inventory-items'), mutate('inventory-overview')])
      onCompleted({ item, alias })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Failed to save item master data.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <DialogContent className="max-w-3xl">
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          <PackagePlus className="h-5 w-5" />
          {isNewItemMode ? 'Register Item and Alias' : 'Register Supplier Alias'}
        </DialogTitle>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="space-y-5">
        {isNewItemMode ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="master-item-number">Item Number</Label>
              <Input id="master-item-number" value={itemNumber} onChange={(event) => setItemNumber(event.target.value)} autoFocus />
            </div>
            <div className="space-y-2">
              <Label htmlFor="master-manufacturer">Manufacturer Key</Label>
              <Input id="master-manufacturer" value={manufacturerKey} onChange={(event) => setManufacturerKey(normalizeKey(event.target.value))} />
            </div>
            <div className="space-y-2 sm:col-span-2">
              <Label htmlFor="master-description">Description</Label>
              <Input id="master-description" value={description} onChange={(event) => setDescription(event.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="master-category">Category</Label>
              <Input
                id="master-category"
                list="master-category-options"
                value={categoryKey}
                onChange={(event) => setCategoryKey(normalizeKey(event.target.value))}
              />
              <datalist id="master-category-options">
                {(masterData?.categories ?? []).map((category) => <option key={category.key} value={category.key}>{category.name}</option>)}
              </datalist>
            </div>
            <div className="space-y-2">
              <Label htmlFor="master-note">Note</Label>
              <Input id="master-note" value={note} onChange={(event) => setNote(event.target.value)} />
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            <Label>Canonical Item</Label>
            <ItemCombobox items={itemOptions} value={itemId} onValueChange={setItemId} placeholder="Select canonical item" />
          </div>
        )}

        <div className="grid gap-4 sm:grid-cols-3">
          <div className="space-y-2">
            <Label htmlFor="master-supplier">Supplier</Label>
            <Select value={selectedSupplierId} onValueChange={(value) => setSupplierId(value === NO_SUPPLIER ? '' : value)}>
              <SelectTrigger id="master-supplier"><SelectValue placeholder="Optional supplier" /></SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_SUPPLIER}>No alias</SelectItem>
                {(masterData?.suppliers ?? []).map((supplier) => (
                  <SelectItem key={supplier.id} value={supplier.id}>{supplier.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="master-alias">Supplier Alias</Label>
            <Input id="master-alias" value={supplierAliasNumber} onChange={(event) => setSupplierAliasNumber(event.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="master-units">Units per Order</Label>
            <Input id="master-units" type="number" min={1} value={unitsPerOrder} onChange={(event) => setUnitsPerOrder(Number(event.target.value) || 1)} />
          </div>
        </div>

        {error && <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>Cancel</Button>
          <Button type="submit" className="gap-2" disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Save
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  )
}

function normalizeKey(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/_/g, '-')
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]+/g, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
}
